// Copyright 2025 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	cli "github.com/openshift-pipelines/tekton-assist/pkg/cli"
)

// mockLightspeedServer returns a test server implementing /v1/query.
func mockLightspeedServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Basic shape validation
		var payload map[string]any
		_ = json.Unmarshal(b, &payload)
		// Return a simple structured response
		resp := map[string]any{
			"response": "TaskRun 'demo' failed due to example error.",
			"analysis": "The container exited with code 1 while running step 'build' due to missing dependency.",
			"solutions": []string{
				"Add the missing dependency to the image or step setup.",
				"Verify network access to fetch dependencies.",
				"Pin versions to ensure reproducibility.",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(mux)
}

func TestE2E_TaskRun_TextOutput(t *testing.T) {
	srv := mockLightspeedServer(t)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	args := []string{
		"taskrun", "diagnose", "demo", "-n", "default",
		"--lightspeed-url", srv.URL,
	}
	root := cli.RootCommand()
	root.SetArgs(args)

	// capture global stdout/stderr since command prints via fmt
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	err := root.ExecuteContext(ctx)
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	_, _ = io.Copy(&buf, rErr)
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, buf.String())
	}

	got := buf.String()
	if !strings.Contains(got, "TaskRun Diagnosis Report") {
		t.Fatalf("missing header in output:\n%s", got)
	}
	if !strings.Contains(got, "Summary:") {
		t.Fatalf("missing Summary section:\n%s", got)
	}
	if !strings.Contains(got, "Solutions:") {
		t.Fatalf("missing Solutions section:\n%s", got)
	}
}

func TestE2E_TaskRun_JSONOutput(t *testing.T) {
	srv := mockLightspeedServer(t)
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	args := []string{
		"taskrun", "diagnose", "demo", "-n", "default",
		"--lightspeed-url", srv.URL,
		"-o", "json",
	}
	root := cli.RootCommand()
	root.SetArgs(args)
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	err := root.ExecuteContext(ctx)
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, rOut)
	_, _ = io.Copy(&buf, rErr)
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, buf.String())
	}
	// Should be valid JSON and include our keys
	var js map[string]any
	if err := json.Unmarshal(buf.Bytes(), &js); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if _, ok := js["response"]; !ok {
		t.Fatalf("missing 'response' field in JSON: %s", buf.String())
	}
}

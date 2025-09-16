package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/openshift-pipelines/tekton-assist/pkg/types"
)

type explainResponse struct {
	Debug    types.TaskRunDebugInfo `json:"debug"`
	Analysis string                 `json:"analysis"`
}

func runCommand(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s %s\nerr: %v\noutput: %s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func tryHTTPGet(url string, timeout time.Duration) (*http.Response, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	return resp, data, nil
}

// TestExplainFailureE2E spins up port-forward to the in-cluster service, patches
// config to disable OpenAI (no API key needed), creates a failing TaskRun, and
// verifies the explainFailure endpoint returns structured debug info.
func TestExplainFailureE2E(t *testing.T) {
	// Basic preflight: ensure kubectl can talk to the cluster.
	if _, err := exec.LookPath("kubectl"); err != nil {
		t.Skip("kubectl not found in PATH; skipping e2e test")
	}
	if out, err := exec.Command("kubectl", "version", "--short").CombinedOutput(); err != nil {
		t.Skipf("kubectl cannot reach cluster: %v; output: %s", err, string(out))
	}

	// Deploy the mock OpenAI-compatible server with ko (required).
	if _, err := exec.LookPath("ko"); err != nil {
		t.Fatalf("ko not found in PATH; install ko and set KO_DOCKER_REPO=kind.local. error: %v", err)
	}
	// Default KO_DOCKER_REPO to kind.local if not set
	if os.Getenv("KO_DOCKER_REPO") == "" {
		_ = os.Setenv("KO_DOCKER_REPO", "kind.local")
	}
	runCommand(t, "ko", "apply", "-BRf", "assets/mock-openai.yaml")
	runCommand(t, "kubectl", "-n", "openshift-pipelines", "rollout", "status", "deploy/mock-openai", "--timeout=180s")
	expected := "mock-analysis"
	// Set base URL to mock service and set provider=ollama to bypass API key requirement.
	runCommand(t, "kubectl", "-n", "openshift-pipelines", "set", "env", "deploy/tekton-assist", "OPENAI_BASE_URL=http://mock-openai.openshift-pipelines.svc.cluster.local:8081/", "PROVIDER=ollama", "DEBUG=true")
	runCommand(t, "kubectl", "-n", "openshift-pipelines", "rollout", "status", "deploy/tekton-assist", "--timeout=180s")
	// Cleanup env overrides to avoid conflicts with manifests in subsequent applies.
	t.Cleanup(func() {
		_ = exec.Command("kubectl", "-n", "openshift-pipelines", "set", "env", "deploy/tekton-assist", "OPENAI_BASE_URL-", "PROVIDER-", "DEBUG-").Run()
		_ = exec.Command("kubectl", "-n", "openshift-pipelines", "rollout", "status", "deploy/tekton-assist", "--timeout=180s").Run()
		_ = exec.Command("kubectl", "-n", "openshift-pipelines", "delete", "-f", "assets/mock-openai.yaml", "--ignore-not-found=true").Run()
	})

	// Apply failing TaskRun.
	runCommand(t, "kubectl", "apply", "-f", "assets/failing-taskrun.yaml")
	t.Cleanup(func() {
		_ = exec.Command("kubectl", "-n", "default", "delete", "taskrun", "failing-tr", "--ignore-not-found=true").Run()
	})

	// Wait for TaskRun to report Succeeded=False.
	runCommand(t, "kubectl", "-n", "default", "wait", "--for=condition=Succeeded=False", "--timeout=240s", "taskrun/failing-tr")

	// Start port-forward to the service.
	pf := exec.Command("kubectl", "-n", "openshift-pipelines", "port-forward", "svc/tekton-assist", "18080:8080")
	pf.Stdout = nil
	pf.Stderr = nil
	if err := pf.Start(); err != nil {
		t.Fatalf("failed to start port-forward: %v", err)
	}
	defer func() { _ = pf.Process.Kill() }()

	// Wait for port-forward to be ready by probing the endpoint.
	baseURL := "http://127.0.0.1:18080"
	deadline := time.Now().Add(30 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for port-forward to become ready")
		}
		resp, _, err := tryHTTPGet(baseURL+"/health", 2*time.Second)
		if err == nil && resp.StatusCode < 500 { // health may not be implemented; ignore errors
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Call explainFailure endpoint.
	url := fmt.Sprintf("%s/taskrun/explainFailure?namespace=default&taskrun=failing-tr", baseURL)
	var resp *http.Response
	var body []byte
	var err error
	deadline = time.Now().Add(45 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timed out calling endpoint, last error: %v", err)
		}
		resp, body, err = tryHTTPGet(url, 5*time.Second)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(750 * time.Millisecond)
	}

	var out explainResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("failed to parse response JSON: %v\nbody: %s", err, string(body))
	}

	if out.Debug.TaskRun != "failing-tr" {
		t.Fatalf("unexpected taskrun: got %q", out.Debug.TaskRun)
	}
	if out.Debug.Namespace != "default" {
		t.Fatalf("unexpected namespace: got %q", out.Debug.Namespace)
	}
	if out.Debug.Succeeded {
		t.Fatalf("expected Succeeded=false, got true")
	}
	if out.Debug.FailedStep.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code; debug=%+v", out.Debug)
	}
	// Assert mocked analysis is returned
	if strings.TrimSpace(out.Analysis) != expected {
		t.Fatalf("unexpected analysis. got=%q want=%q", out.Analysis, expected)
	}
}

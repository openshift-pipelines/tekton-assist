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

package analysis

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type LLM interface {
	Analyze(ctx context.Context, input string) (string, error)
}

// OpenAIConfig holds configuration for the OpenAI-backed LLM.
type OpenAIConfig struct {
	APIKey         string
	Provider       string
	Model          string
	Temperature    float32
	MaxTokens      int
	BaseURL        string
	RequestTimeout time.Duration
	Debug          bool
}

type openAILLM struct {
	client    openai.Client
	model     string
	temp      float32
	maxTokens int
	debug     bool
}

// NewOpenAILLM constructs an LLM that talks to OpenAI's chat completions.
func NewOpenAILLM(cfg OpenAIConfig) (LLM, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	// Secrets mounted via env may include trailing newlines; trim to avoid invalid Authorization header
	apiKey = strings.TrimSpace(apiKey)
	if cfg.Provider != "ollama" && apiKey == "" {
		log.Fatal("API key is required for provider: ", cfg.Provider)
	}

	// Build client options
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.RequestTimeout > 0 {
		hc := &http.Client{Timeout: cfg.RequestTimeout}
		opts = append(opts, option.WithHTTPClient(hc))
	}

	c := openai.NewClient(opts...)
	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &openAILLM{
		client:    c,
		model:     model,
		temp:      cfg.Temperature,
		maxTokens: cfg.MaxTokens,
		debug:     cfg.Debug,
	}, nil
}

func (o *openAILLM) Analyze(ctx context.Context, input string) (string, error) {
	if o.debug {
		log.Printf("llm: model=%s prompt_len=%d", o.model, len(input))
	}
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a Tekton TaskRun debugging expert. Provide concise, actionable diagnosis and fixes."),
			openai.UserMessage(input),
		},
		Model: openai.ChatModel(o.model),
	}
	// Note: temperature and max tokens omitted for now to avoid param.Opt types

	resp, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		if o.debug {
			log.Printf("llm: error=%v", err)
		}
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty completion choices")
	}
	out := resp.Choices[0].Message.Content
	if o.debug {
		log.Printf("llm: response_len=%d", len(out))
	}
	return out, nil
}

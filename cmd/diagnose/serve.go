package main

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift-pipelines/tekton-assist/pkg/analysis"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the tekton-assist HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.New(os.Stdout, "tekton-assist ", log.LstdFlags|log.Lshortfile)

		// Override defaults/flags from environment variables when provided (populated via ConfigMap)
		if v := os.Getenv("PROVIDER"); v != "" {
			cfg.Provider = v
		}
		if v := os.Getenv("OPENAI_MODEL"); v != "" {
			cfg.OpenAIModel = v
		}
		if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
			cfg.OpenAIBase = v
		}
		if v := os.Getenv("OPENAI_TEMPERATURE"); v != "" {
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				cfg.Temperature = float32(f)
			}
		}
		if v := os.Getenv("OPENAI_MAX_TOKENS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				cfg.MaxTokens = n
			}
		}
		if v := os.Getenv("OPENAI_TIMEOUT"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				cfg.Timeout = d
			}
		}
		if v := os.Getenv("DEBUG"); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				cfg.Debug = b
			}
		}

		llm, err := analysis.NewOpenAILLM(analysis.OpenAIConfig{
			Provider:       cfg.Provider,
			Model:          cfg.OpenAIModel,
			BaseURL:        cfg.OpenAIBase,
			Temperature:    cfg.Temperature,
			MaxTokens:      cfg.MaxTokens,
			RequestTimeout: cfg.Timeout,
			Debug:          cfg.Debug,
		})
		if err != nil {
			logger.Printf("warning: OpenAI LLM disabled: %v", err)
		}
		srv := NewHTTPServer(cfg.Addr, logger, llm)

		var wg sync.WaitGroup
		srv.startListener(&wg)
		wg.Wait()
	},
}

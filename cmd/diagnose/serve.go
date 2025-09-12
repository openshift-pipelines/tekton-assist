package main

import (
	"log"
	"os"
	"sync"

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
		llm, err := analysis.NewOpenAILLM(analysis.OpenAIConfig{
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

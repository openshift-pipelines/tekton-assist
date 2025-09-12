package main

import (
	"time"

	"github.com/spf13/cobra"
)

type Config struct {
	Addr string
	// LLM config
	OpenAIModel string
	OpenAIBase  string
	Temperature float32
	MaxTokens   int
	Timeout     time.Duration
	Debug       bool
}

var (
	rootCmd = &cobra.Command{Use: "diagnose", Short: "Tekton TaskRun failure explainer"}
	cfg     = &Config{}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cfg.Addr, "addr", ":8080", "HTTP listen address")
	rootCmd.PersistentFlags().StringVar(&cfg.OpenAIModel, "openai-model", "gpt-4o-mini", "OpenAI model name")
	rootCmd.PersistentFlags().StringVar(&cfg.OpenAIBase, "openai-base-url", "", "Optional OpenAI-compatible base URL")
	rootCmd.PersistentFlags().Float32Var(&cfg.Temperature, "openai-temperature", 0.2, "OpenAI sampling temperature")
	rootCmd.PersistentFlags().IntVar(&cfg.MaxTokens, "openai-max-tokens", 400, "OpenAI max output tokens")
	rootCmd.PersistentFlags().DurationVar(&cfg.Timeout, "openai-timeout", 30*time.Second, "OpenAI request timeout")
	rootCmd.PersistentFlags().BoolVar(&cfg.Debug, "debug", false, "Enable verbose logging")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func main() { Execute() }

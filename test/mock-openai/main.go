package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type choiceMessage struct {
	Content string `json:"content"`
}

type choice struct {
	Message choiceMessage `json:"message"`
}

type completionResponse struct {
	Choices []choice `json:"choices"`
}

func main() {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		handleCompletion(w, r)
	})

	// OpenAI-compatible chat completions path
	mux.HandleFunc("/v1/chat/completions", handleCompletion)

	addr := ":8081"
	log.Printf("mock-openai listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux}
	log.Fatal(srv.ListenAndServe())
}

func handleCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	content := os.Getenv("MOCK_ANALYSIS")
	if content == "" {
		content = "mock-analysis"
	}
	resp := completionResponse{Choices: []choice{{Message: choiceMessage{Content: content}}}}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

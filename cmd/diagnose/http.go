package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/openshift-pipelines/tekton-assist/pkg/analysis"
	"github.com/openshift-pipelines/tekton-assist/pkg/inspector"
)

// HandlerFunc defines a generic HTTP handler function type
type HandlerFunc func(w http.ResponseWriter, r *http.Request)

// httpServer wraps http.Server and modular handlers
type httpServer struct {
	*http.Server
	httpServerEndpoint string
	log                *log.Logger
	handlers           map[string]HandlerFunc
	llm                analysis.LLM
}

// NewHTTPServer creates a new httpServer with modular handlers
func NewHTTPServer(endpoint string, log *log.Logger, llm analysis.LLM) *httpServer {
	h := &httpServer{
		httpServerEndpoint: endpoint,
		log:                log,
		handlers:           make(map[string]HandlerFunc),
		llm:                llm,
	}

	h.registerHandlers()
	h.initServer()
	return h
}

// registerHandlers registers all HTTP endpoints
func (h *httpServer) registerHandlers() {
	h.handlers["/taskrun/explainFailure"] = h.handleExplainFailure
	// Add more endpoints here if needed
}

// initServer wires handlers, metrics, CORS, and creates http.Server
func (h *httpServer) initServer() {
	mux := http.NewServeMux()
	for path, handler := range h.handlers {
		// Wrap with Prometheus metrics and CORS
		// handler := promhttp.InstrumentHandlerDuration(server.MetricLatency, mux)
		// handler = promhttp.InstrumentHandlerCounter(server.RequestsCount, handler)
		// handler = cors.Default().Handler(handler)
		mux.HandleFunc(path, handler)
	}

	h.Server = &http.Server{
		Addr:         h.httpServerEndpoint,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// handleDiagnose handles the /taskrun/diagnose endpoint
func (h *httpServer) handleExplainFailure(w http.ResponseWriter, r *http.Request) {
	taskrunName := r.URL.Query().Get("taskrun")
	namespace := r.URL.Query().Get("namespace")
	if taskrunName == "" || namespace == "" {
		http.Error(w, "missing taskrun name or namespace", http.StatusBadRequest)
		return
	}

	h.log.Printf("Diagnose request received: taskrun name=%s, namespace=%s", taskrunName, namespace)

	ins, err := inspector.NewInspector()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create inspector: %v", err), http.StatusInternalServerError)
		return
	}

	result, err := ins.InspectTaskRun(r.Context(), namespace, taskrunName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to inspect taskrun: %v", err), http.StatusInternalServerError)
		return
	}

	// Optionally ask LLM for diagnosis
	var analysisText string
	if h.llm != nil {
		prompt := analysis.BuildTaskRunPrompt(result)
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		if out, err := h.llm.Analyze(ctx, prompt); err == nil {
			analysisText = out
		} else {
			h.log.Printf("LLM analyze failed: %v", err)
		}
	}

	// DEMO: pretty text output
	/*w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(analysis.RenderPrettyReportANSI(result, analysisText)))
	return*/

	// Original JSON response (commented out for demo; keep to rollback easily)
	type response struct {
		Debug    interface{} `json:"debug"`
		Analysis string      `json:"analysis,omitempty"`
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response{Debug: result, Analysis: analysisText}); err != nil {
		h.log.Printf("Failed to encode response: %v", err)
	}
}

// startListener starts the HTTP server with graceful shutdown
func (h *httpServer) startListener(wg *sync.WaitGroup) {
	h.log.Printf("HTTP server listening on %s", h.httpServerEndpoint)

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		if err := h.Shutdown(context.Background()); err != nil {
			h.log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
		h.log.Printf("stopped http server")
	}()

	wg.Add(1)
	go func() {
		if err := h.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			h.log.Fatalf("HTTP server failed: %v", err)
		}
		<-idleConnsClosed
		wg.Done()
		h.log.Printf("http server shutdown")
	}()
}

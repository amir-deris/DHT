package server

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/amirderis/DHT/internal/config"
)

type HTTPServer struct {
	cfg       *config.Config
	server    *http.Server
	readyFlag atomic.Bool
}

func NewHTTPServer(cfg *config.Config) *HTTPServer {
	mux := http.NewServeMux()
	s := &HTTPServer{cfg: cfg}
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)

	s.server = &http.Server{
		Addr:         cfg.BindAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set ready true after initialization; in Phase 1.1 we flip immediately
	s.readyFlag.Store(true)

	return s
}

func (s *HTTPServer) Start() error {
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}

func (s *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	if !s.readyFlag.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintln(w, "not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ready")
}

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/amirderis/DHT/internal/config"
	"github.com/amirderis/DHT/internal/storage"
	"github.com/amirderis/DHT/internal/ring"
	"github.com/amirderis/DHT/pkg/api"
)

type HTTPServer struct {
	cfg       *config.Config
	server    *http.Server
	readyFlag atomic.Bool
	storage   storage.Engine
	ring      *ring.Ring
}

func NewHTTPServer(cfg *config.Config) *HTTPServer {
	mux := http.NewServeMux()
	s := &HTTPServer{
		cfg:     cfg,
		storage: storage.NewInMemory(),
		ring:    ring.New(20), // 20 virtual nodes per physical node
	}

	// Initialize ring with this node
	s.ring.AddNode(ring.NodeID(cfg.NodeID), cfg.BindAddr)

	// Health and readiness endpoints
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)

	// KV API endpoints
	mux.HandleFunc("/kv/", s.handleKV)

	s.server = &http.Server{
		Addr:         cfg.BindAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set ready true after initialization
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

// handleKV routes GET/PUT/DELETE requests to appropriate handlers
func (s *HTTPServer) handleKV(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/kv/"):]
	if key == "" {
		s.writeError(w, http.StatusBadRequest, "key cannot be empty")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r, key)
	case http.MethodPut:
		s.handlePut(w, r, key)
	case http.MethodDelete:
		s.handleDelete(w, r, key)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleGet(w http.ResponseWriter, _ *http.Request, key string) {
	value, found := s.storage.Get(key)

	response := api.GetResponse{
		Key:   key,
		Value: value,
		Found: found,
	}

	if found {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	s.writeJSON(w, response)
}

func (s *HTTPServer) handlePut(w http.ResponseWriter, r *http.Request, key string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	if err := s.storage.Put(key, body); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to store value")
		return
	}

	response := api.PutResponse{
		Version: map[string]uint64{s.cfg.NodeID: 1}, // Placeholder for vector clock
	}

	w.WriteHeader(http.StatusOK)
	s.writeJSON(w, response)
}

func (s *HTTPServer) handleDelete(w http.ResponseWriter, _ *http.Request, key string) {
	if err := s.storage.Delete(key); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Log error but don't write to response as headers may already be sent
		fmt.Printf("failed to encode JSON response: %v\n", err)
	}
}

func (s *HTTPServer) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errorResp := map[string]string{"error": message}
	json.NewEncoder(w).Encode(errorResp)
}

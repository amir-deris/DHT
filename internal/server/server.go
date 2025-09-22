package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/amirderis/DHT/internal/config"
	"github.com/amirderis/DHT/internal/ring"
	"github.com/amirderis/DHT/internal/storage"
	"github.com/amirderis/DHT/pkg/api"
)

const (
	readConsistencyHeader  = "X-Consistency-R"
	writeConsistencyHeader = "X-Consistency-W"
)

type HTTPServer struct {
	cfg       *config.Config
	server    *http.Server
	readyFlag atomic.Bool
	storage   storage.Engine
	ring      *ring.Ring
	client    *http.Client
}

func NewHTTPServer(cfg *config.Config) *HTTPServer {
	mux := http.NewServeMux()
	s := &HTTPServer{
		cfg:     cfg,
		storage: storage.NewInMemory(),
		ring:    ring.New(20), // 20 virtual nodes per physical node
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Initialize ring with this node
	s.ring.AddNode(ring.NodeID(cfg.NodeID), cfg.BindAddr)

	// Health and readiness endpoints
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)

	// KV API endpoints
	mux.HandleFunc("/kv/", s.handleKV)

	// Internal storage endpoints
	mux.HandleFunc("/internal/storage/", s.handleInternalStorage)

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

// handleKV routes GET/PUT/DELETE requests for a key to appropriate handlers
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
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed: " + r.Method)
	}
}

func (s *HTTPServer) handleGet(w http.ResponseWriter, r *http.Request, key string) {
	readQuorum := s.getQuorumFromHeader(r, readConsistencyHeader, s.cfg.ReadQuorum)

	preferenceList, err := s.ring.GetPreferenceList(key, s.cfg.ReplicationFactor)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get preference list for key: "+key)
		return
	}

	// If we only have one node or read quorum=1, just read locally
	if len(preferenceList) == 1 || readQuorum == 1 {
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
		return
	}

	// Read from multiple nodes
	responses := s.readFromNodes(key, preferenceList, readQuorum)
	if len(responses) < readQuorum {
		message := fmt.Sprintf("expected %d replicas, got %d", readQuorum, len(responses))
		s.writeError(w, http.StatusServiceUnavailable, message)
		return
	}

	// For now, return the first successful response
	// TODO: Implement conflict resolution in Phase 3
	var response api.GetResponse
	for _, resp := range responses {
		if resp.Found {
			response = resp
			break
		}
	}
	if response.Found {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	s.writeJSON(w, response)
}

func (s *HTTPServer) handlePut(w http.ResponseWriter, r *http.Request, key string) {
	writeQuorum := s.getQuorumFromHeader(r, writeConsistencyHeader, s.cfg.WriteQuorum)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	preferenceList, err := s.ring.GetPreferenceList(key, s.cfg.ReplicationFactor)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to get preference list for key: "+key)
		return
	}

	// Create version (placeholder for vector clock)
	version := map[string]uint64{s.cfg.NodeID: 1}

	// If we only have one node or write quorum=1, just write locally
	if len(preferenceList) == 1 || writeQuorum == 1 {
		if err := s.storage.Put(key, body); err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to store value")
			return
		}

		response := api.PutResponse{Version: version}
		w.WriteHeader(http.StatusOK)
		s.writeJSON(w, response)
		return
	}

	// Write to multiple nodes
	successCount := s.writeToNodes(key, body, version, preferenceList, writeQuorum)
	if successCount < writeQuorum {
		s.writeError(w, http.StatusServiceUnavailable, "insufficient replicas available for write quorum for key: "+key)
		return
	}

	response := api.PutResponse{Version: version}
	w.WriteHeader(http.StatusOK)
	s.writeJSON(w, response)
}

// writeToNodes writes to multiple nodes and returns success count
func (s *HTTPServer) writeToNodes(key string, value []byte, version map[string]uint64, prefList []ring.NodeID, writeQuorum int) int {
	successCount := 0

	for _, nodeID := range prefList {
		if successCount >= writeQuorum {
			break
		}

		// If it's this node, write locally
		if nodeID == ring.NodeID(s.cfg.NodeID) {
			if err := s.storage.Put(key, value); err == nil {
				successCount++
			} else {
				fmt.Printf("failed to write to local node %s for key: %s, error: %v\n", s.cfg.NodeID, key, err)
			}
			continue
		}

		// Write to remote node
		address, exists := s.ring.GetNodeAddress(nodeID)
		if !exists {
			fmt.Printf("node %s not found in ring for key: %s\n", nodeID, key)
			continue
		}
		if err := s.writeToRemoteNode(address, key, value, version); err == nil {
			successCount++
		} else {
			fmt.Printf("failed to write to remote node %s for key: %s, error: %v\n", address, key, err)
		}
	}
	return successCount
}

func (s *HTTPServer) writeToRemoteNode(address, key string, value []byte, version map[string]uint64) error {
	req := api.ReplicateRequest{
		Key:     key,
		Value:   value,
		Version: version,
	}
	var jsonData bytes.Buffer
	if err := json.NewEncoder(&jsonData).Encode(req); err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s/internal/storage/%s", address, key)
	resp, err := s.client.Post(url, "application/json", strings.NewReader(jsonData.String()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote node %s returned status %d", address, resp.StatusCode)
	}

	var result api.ReplicateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("remote node %s failed to store value", address)
	}

	return nil
}

func (s *HTTPServer) handleDelete(w http.ResponseWriter, _ *http.Request, key string) {
	if err := s.storage.Delete(key); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to delete key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleInternalStorage(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/internal/storage/"):]
	if key == "" {
		s.writeError(w, http.StatusBadRequest, "key cannot be empty")
		return
	}

	switch r.Method {
	case http.MethodGet:
		value, found := s.storage.Get(key)
		response := api.ReplicateGetResponse{
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
	case http.MethodPost:
		var req api.ReplicateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := s.storage.Put(key, req.Value); err != nil {
			response := api.ReplicateResponse{
				Success: false,
				Error:   "failed to store value",
			}
			w.WriteHeader(http.StatusInternalServerError)
			s.writeJSON(w, response)
			return
		}

		response := api.ReplicateResponse{Success: true}
		w.WriteHeader(http.StatusOK)
		s.writeJSON(w, response)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed: "+r.Method)
	}
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

func (s *HTTPServer) getQuorumFromHeader(r *http.Request, headerName string, defaultValue int) int {
	if headerValue := r.Header.Get(headerName); headerValue != "" {
		var quorum int
		quorum, err := strconv.Atoi(headerValue)
		if err == nil && quorum > 0 {
			return quorum
		}
	}
	return defaultValue
}

func (s *HTTPServer) readFromNodes(key string, prefList []ring.NodeID, readQuorum int) []api.GetResponse {
	responses := make([]api.GetResponse, 0, len(prefList))

	for _, nodeID := range prefList {
		if len(responses) >= readQuorum {
			break
		}

		// If it's this node, read locally
		if nodeID == ring.NodeID(s.cfg.NodeID) {
			value, found := s.storage.Get(key)
			responses = append(responses, api.GetResponse{
				Key:   key,
				Value: value,
				Found: found,
			})
			continue
		}

		// Read from remote node
		address, exists := s.ring.GetNodeAddress(nodeID)
		if !exists {
			continue
		}

		resp, err := s.readFromRemoteNode(address, key)
		if err == nil {
			responses = append(responses, resp)
		}
	}
	return responses
}

func (s *HTTPServer) readFromRemoteNode(address, key string) (api.GetResponse, error) {
	url := fmt.Sprintf("http://%s/internal/storage/%s", address, key)
	resp, err := s.client.Get(url)
	if err != nil {
		return api.GetResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return api.GetResponse{}, fmt.Errorf("remote node returned status %d", resp.StatusCode)
	}

	var result api.ReplicateGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.GetResponse{}, err
	}
	return api.GetResponse{
		Key:   result.Key,
		Value: result.Value,
		Found: result.Found,
	}, nil
}

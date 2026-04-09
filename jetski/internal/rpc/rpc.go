package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/armorclaw/jetski/internal/approval"
)

type Server struct {
	startTime time.Time
	sessions  map[string]struct{}
	mu        sync.RWMutex
	counter   atomic.Int64
	ac        *approval.ApprovalClient
}

func NewServer(ac *approval.ApprovalClient) *Server {
	return &Server{
		startTime: time.Now(),
		sessions:  make(map[string]struct{}),
		ac:        ac,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc/status", s.handleStatus)
	mux.HandleFunc("/rpc/session/create", s.handleSessionCreate)
	mux.HandleFunc("/rpc/session/close", s.handleSessionClose)
	mux.HandleFunc("/rpc/health", s.handleHealth)
	if s.ac != nil {
		approval.RegisterApprovalHandlers(mux, s.ac)
	}
	return mux
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	active := len(s.sessions)
	s.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_sessions": active,
		"engine_health":   "ok",
	})
}

func (s *Server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := fmt.Sprintf("session-%d", s.counter.Add(1))

	s.mu.Lock()
	s.sessions[id] = struct{}{}
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"id": id,
	})
}

func (s *Server) handleSessionClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	s.mu.Lock()
	_, exists := s.sessions[req.ID]
	if !exists {
		s.mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "session not found"})
		return
	}
	delete(s.sessions, req.ID)
	s.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "closed"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(s.startTime).Seconds()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"uptime": uptime,
	})
}

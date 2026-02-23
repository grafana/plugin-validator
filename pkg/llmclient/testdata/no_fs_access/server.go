package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	mu       sync.RWMutex
	cache    map[string]CacheEntry
	handlers map[string]http.HandlerFunc
}

type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

func NewServer() *Server {
	return &Server{
		cache:    make(map[string]CacheEntry),
		handlers: make(map[string]http.HandlerFunc),
	}
}

func (s *Server) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.cache[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

func (s *Server) Set(key string, value interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[key] = CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) HandleData(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key parameter", http.StatusBadRequest)
		return
	}

	if val, ok := s.Get(key); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(val)
		return
	}

	http.Error(w, fmt.Sprintf("key %q not found", key), http.StatusNotFound)
}

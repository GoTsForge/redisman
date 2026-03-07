package server

import "sync"

type Server struct {
	mu    *sync.RWMutex
	store map[string]string
}

func NewServer() *Server {
	var mu sync.RWMutex
	store := make(map[string]string)

	return &Server{
		mu:    &mu,
		store: store,
	}
}

func (s *Server) Set(key string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[key] = value
}

func (s *Server) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	storeValue, ok := s.store[key]
	if !ok {
		return "", false
	}

	return storeValue, true
}

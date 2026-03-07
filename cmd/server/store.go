package server

import (
	"sync"
	"time"
)

type Value struct {
	Value  string
	Expiry time.Time
}

type Server struct {
	mu    *sync.RWMutex
	store map[string]Value
}

func NewServer() *Server {
	var mu sync.RWMutex
	store := make(map[string]Value)

	return &Server{
		mu:    &mu,
		store: store,
	}
}

func (s *Server) Set(key string, value string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[key] = Value{
		Value:  value,
		Expiry: expiry,
	}
}

func (s *Server) Get(key string) (Value, bool) {
	s.mu.RLock()

	storeValue, ok := s.store[key]

	if !ok {
		s.mu.RUnlock()
		return Value{}, false
	}

	// if the value has expired, delete it and return missing
	if !storeValue.Expiry.IsZero() && time.Now().After(storeValue.Expiry) {
		// REDIS actually doesn't delete keys - to avoid extra writes while reading values. The deleted values are then cleaned up later.
		// But we will...
		// we first unlock the RLock
		s.mu.RUnlock()
		// since we're now mutating the map, we need a write lock
		s.mu.Lock()
		delete(s.store, key)
		s.mu.Unlock()
		return Value{}, false
	} else {
		s.mu.RUnlock()
	}

	return storeValue, true
}

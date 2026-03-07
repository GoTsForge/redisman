package server

import (
	"fmt"
	"sync"
	"time"
)

type ValueType int

const (
	TypeString ValueType = iota
	TypeList
)

type Value struct {
	Type        ValueType
	StringValue string
	ListValue   []string
	Expiry      time.Time
}

func NewStringValue(val string, expiry time.Time) Value {
	return Value{Type: TypeString, StringValue: val, Expiry: expiry}
}

func NewListEntry(values []string) Value {
	return Value{Type: TypeList, ListValue: values}
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

	valToStore := NewStringValue(value, expiry)
	s.store[key] = valToStore
}

func (s *Server) RPush(key string, vals ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	storedVal, ok := s.store[key]
	if !ok {
		// we need to create a new list
		s.store[key] = NewListEntry(vals)
		return len(vals), nil
	}

	// the key already exists, we need to check if this is actually a list

	if storedVal.Type != TypeList {
		return 0, fmt.Errorf("WRONGTYPE key is not a list")
	}

	storedVal.ListValue = append(storedVal.ListValue, vals...)
	s.store[key] = storedVal

	return len(storedVal.ListValue), nil
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

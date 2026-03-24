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
	// waiters is a map of key -> channels used for blocking operations
	// multiple BLPOP meaning multiple goroutines can wait on the same key - hence we need a slice of channels
	waiters map[string][]chan struct{}
}

func NewServer() *Server {
	var mu sync.RWMutex
	store := make(map[string]Value)
	waiters := make(map[string][]chan struct{})

	return &Server{
		mu:      &mu,
		store:   store,
		waiters: waiters,
	}
}

func (s *Server) Set(key string, value string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	valToStore := NewStringValue(value, expiry)
	s.store[key] = valToStore
}

func (s *Server) signalFirstWaiter(key string) {
	firstWaiterForKey := s.waiters[key][0]
	s.waiters[key] = s.waiters[key][1:]

	// signal the channel that a value is pushed
	firstWaiterForKey <- struct{}{}
}

func (s *Server) LPush(key string, vals []string) (int, error) {
	s.mu.Lock()

	storedVal, ok := s.store[key]
	if !ok {
		// we need to create a new list
		s.store[key] = NewListEntry(vals)
		if len(s.waiters[key]) > 0 {
			s.signalFirstWaiter(key)
		}

		s.mu.Unlock()
		return len(vals), nil
	}

	// the key already exists, we need to check if this is actually a list
	if storedVal.Type != TypeList {
		return 0, fmt.Errorf("WRONGTYPE key is not a list")
	}

	if len(s.waiters[key]) > 0 {
		s.signalFirstWaiter(key)
	}

	storedVal.ListValue = append(vals, storedVal.ListValue...)
	s.store[key] = storedVal

	s.mu.Unlock()
	return len(storedVal.ListValue), nil
}

func (s *Server) LRange(key string, start int, stop int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	storedVal, exists := s.store[key]
	if !exists {
		return []string{}, nil
	}

	if storedVal.Type != TypeList {
		return []string{}, fmt.Errorf("WRONGTYPE key is not of list type")
	}

	length := len(storedVal.ListValue)

	if start < 0 {
		start = length + start
	}

	if stop < 0 {
		stop = length + stop
	}

	// start could still be less than 0 if the user input a very large negative number, we don't care in that case
	if start < 0 {
		start = 0
	}

	if start > stop {
		return []string{}, nil
	}

	if start >= length {
		return []string{}, nil
	}

	if stop >= length {
		stop = length - 1
	}

	resultLength := stop - start + 1
	result := make([]string, resultLength)

	// why copy - to prevent external memory mutation, should be optional but better be safe.
	// otherwise slicing does not return a new slice, it creates a header pointing to the same slice underneath.
	copy(result, storedVal.ListValue[start:stop+1])

	return result, nil
}

func (s *Server) RPush(key string, vals []string) (int, error) {
	s.mu.Lock()

	storedVal, ok := s.store[key]
	if !ok {
		// we need to create a new list
		s.store[key] = NewListEntry(vals)
		if len(s.waiters[key]) > 0 {
			s.signalFirstWaiter(key)
		}

		s.mu.Unlock()
		return len(vals), nil
	}

	// the key already exists, we need to check if this is actually a list
	if storedVal.Type != TypeList {
		return 0, fmt.Errorf("WRONGTYPE key is not a list")
	}

	if len(s.waiters[key]) > 0 {
		s.signalFirstWaiter(key)
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

func (s *Server) ListLen(key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, exists := s.store[key]
	if !exists {
		return 0, nil
	}

	if val.Type != TypeList {
		return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return len(val.ListValue), nil
}

func (s *Server) ListPop(key string, numValuesToRemove int) ([]string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, exists := s.store[key]
	if !exists {
		return []string{}, false, nil
	}

	if val.Type != TypeList {
		return []string{}, false, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	if len(val.ListValue) == 0 {
		return []string{}, false, nil
	}

	if numValuesToRemove > len(val.ListValue) {
		numValuesToRemove = len(val.ListValue)
	}

	poppedValues := make([]string, numValuesToRemove)
	copy(poppedValues, val.ListValue[:numValuesToRemove])

	// We do this because we want to make the underlying array elements "zero" so that GC can pick them up immediately
	for i := 0; i < numValuesToRemove; i++ {
		val.ListValue[i] = ""
	}

	val.ListValue = val.ListValue[numValuesToRemove:]

	s.store[key] = val

	return poppedValues, true, nil
}

func (s *Server) cleanup(keys []string, notifyChan chan struct{}) {
	for _, key := range keys {
		for idx, waiter := range s.waiters[key] {
			if waiter == notifyChan {
				s.waiters[key] = append(s.waiters[key][:idx], s.waiters[key][idx+1:]...)

				if len(s.waiters[key]) == 0 {
					delete(s.waiters, key)
				}

				break
			}
		}
	}
}

func (s *Server) BListPop(keys []string, timeout time.Duration) ([]string, bool, error) {
	var deadline <-chan time.Time
	if timeout > time.Duration(0) {
		deadline = time.After(timeout)
	}

	notifyChan := make(chan struct{}, 1)

	for {
		s.mu.Lock()
		s.cleanup(keys, notifyChan)

		for _, key := range keys {
			val := s.store[key]
			if val.Type == TypeList && len(val.ListValue) > 0 {
				poppedVal := val.ListValue[0]
				val.ListValue = val.ListValue[1:]
				s.store[key] = val

				s.mu.Unlock()
				return []string{key, poppedVal}, true, nil
			}
		}

		for _, key := range keys {
			// register the waiters for this key
			s.waiters[key] = append(s.waiters[key], notifyChan)
		}

		s.mu.Unlock()

		select {
		case <-notifyChan:
			// a signal for any key
			continue
		case <-deadline:
			s.mu.Lock()
			// clean up notifyChan from ALL waiters for ALL keys
			s.cleanup(keys, notifyChan)
			s.mu.Unlock()
		}

		return []string{}, false, nil
	}
}

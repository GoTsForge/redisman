package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/gotsforge/redisman/cmd/constants"
	"github.com/gotsforge/redisman/cmd/utils"
)

type ValueType int

const (
	TypeString ValueType = iota
	TypeList
	TypeStream
)

type Entry struct {
	ID   utils.EntryId
	Data map[string]string
}

type Value struct {
	Type        ValueType
	StringValue string
	ListValue   []string
	Entries     []Entry
	Expiry      time.Time
}

type Stream struct {
	Key     string
	Entries []Entry
}

func NewStringValue(val string, expiry time.Time) Value {
	return Value{Type: TypeString, StringValue: val, Expiry: expiry}
}

func NewListEntry(values []string) Value {
	return Value{Type: TypeList, ListValue: values}
}

func NewStream(entries []Entry) Value {
	return Value{
		Type:    TypeStream,
		Entries: entries,
	}
}

type WaiterMap map[string][]chan struct{}

type Server struct {
	mu    *sync.RWMutex
	store map[string]Value
	// blPopWaiters is a map of key -> channels used for blocking operations
	// multiple BLPOP meaning multiple goroutines can wait on the same key - hence we need a slice of channels
	blPopWaiters WaiterMap
	// blXReadWaiters is a map of key -> channels used for blocking operations
	// multiple XREAD BLOCK meaning multiple goroutines can wait on the same key - hence we need a slice of channels
	blXReadWaiters WaiterMap
}

func NewServer() *Server {
	var mu sync.RWMutex
	store := make(map[string]Value)
	listWaiters := make(map[string][]chan struct{})
	streamWaiters := make(map[string][]chan struct{})

	return &Server{
		mu:             &mu,
		store:          store,
		blPopWaiters:   listWaiters,
		blXReadWaiters: streamWaiters,
	}
}

func (s *Server) Set(key string, value string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	valToStore := NewStringValue(value, expiry)
	s.store[key] = valToStore
}

func (s *Server) signalFirstWaiter(key string) {
	firstWaiterForKey := s.blPopWaiters[key][0]
	s.blPopWaiters[key] = s.blPopWaiters[key][1:]

	// signal the channel that a value is pushed
	firstWaiterForKey <- struct{}{}
}

func (s *Server) LPush(key string, vals []string) (int, error) {
	s.mu.Lock()

	storedVal, ok := s.store[key]
	if !ok {
		// we need to create a new list
		s.store[key] = NewListEntry(vals)
		if len(s.blPopWaiters[key]) > 0 {
			s.signalFirstWaiter(key)
		}

		s.mu.Unlock()
		return len(vals), nil
	}

	// the key already exists, we need to check if this is actually a list
	if storedVal.Type != TypeList {
		s.mu.Unlock()
		return 0, fmt.Errorf("WRONGTYPE key is not a list")
	}

	if len(s.blPopWaiters[key]) > 0 {
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
		if len(s.blPopWaiters[key]) > 0 {
			s.signalFirstWaiter(key)
		}

		s.mu.Unlock()
		return len(vals), nil
	}

	// the key already exists, we need to check if this is actually a list
	if storedVal.Type != TypeList {
		s.mu.Unlock()
		return 0, fmt.Errorf("WRONGTYPE key is not a list")
	}

	if len(s.blPopWaiters[key]) > 0 {
		s.signalFirstWaiter(key)
	}

	storedVal.ListValue = append(storedVal.ListValue, vals...)
	s.store[key] = storedVal

	s.mu.Unlock()
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

func (s *Server) cleanupWaiters(keys []string, notifyChan chan struct{}, waiters WaiterMap) {
	for _, key := range keys {
		for idx, waiter := range waiters[key] {
			if waiter == notifyChan {
				waiters[key] = append(waiters[key][:idx], waiters[key][idx+1:]...)

				if len(waiters[key]) == 0 {
					delete(waiters, key)
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
		s.cleanupWaiters(keys, notifyChan, s.blPopWaiters)

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
			s.blPopWaiters[key] = append(s.blPopWaiters[key], notifyChan)
		}

		s.mu.Unlock()

		select {
		case <-notifyChan:
			// a signal for any key
			continue
		case <-deadline:
			s.mu.Lock()
			// clean up notifyChan from ALL waiters for ALL keys
			s.cleanupWaiters(keys, notifyChan, s.blPopWaiters)
			s.mu.Unlock()
		}

		return []string{}, false, nil
	}
}

// returns timestamp and the sequenceNumber
func resolveEntryId(parsedEntryId utils.ParsedEntryId, lastEntry *Entry) (int, int) {
	var resolvedTimestamp int = parsedEntryId.Timestamp
	var resolvedSequenceNumber int = parsedEntryId.SequenceNumber

	if lastEntry == nil {
		// no history of an entry
		if parsedEntryId.IsTimestampAutoGen {
			resolvedTimestamp = int(time.Now().UnixMilli())
		}

		if parsedEntryId.IsSequenceNumberAutoGen {
			if resolvedTimestamp == 0 {
				resolvedSequenceNumber = 1
			} else {
				resolvedSequenceNumber = 0
			}
		}

		return resolvedTimestamp, resolvedSequenceNumber
	}

	if parsedEntryId.IsTimestampAutoGen {
		// get the max of the last entry timestamp and the current timestamp that the clock shows to prevent clock skew
		resolvedTimestamp = max(int(time.Now().UnixMilli()), lastEntry.ID.Timestamp)
	}

	if parsedEntryId.IsSequenceNumberAutoGen {
		if resolvedTimestamp == lastEntry.ID.Timestamp {
			resolvedSequenceNumber = lastEntry.ID.SequenceNumber + 1
		} else {
			if resolvedTimestamp == 0 {
				resolvedSequenceNumber = 1
			} else {
				resolvedSequenceNumber = 0
			}
		}
	}

	return resolvedTimestamp, resolvedSequenceNumber
}

func (s *Server) lookUpLastEntry(key string) (*Entry, error) {
	streamValue, exists := s.store[key]

	if !exists {
		return nil, nil
	}

	if streamValue.Type != TypeStream {
		return nil, fmt.Errorf(constants.ERR_WRONGTYPE_OPERATION)
	}

	if len(streamValue.Entries) == 0 {
		return nil, nil
	}

	return &streamValue.Entries[len(streamValue.Entries)-1], nil
}

func validateResolvedEntryId(timestamp int, sequenceNumber int, lastEntry *Entry) error {
	if timestamp == 0 && sequenceNumber == 0 {
		return fmt.Errorf(constants.ERR_INVALID_ID_MUST_BE_GREATER_XADD)
	}

	if lastEntry != nil {

		if timestamp < lastEntry.ID.Timestamp {
			return fmt.Errorf(constants.ERR_INVALID_ID_XADD_SMALLER)
		}

		if timestamp == lastEntry.ID.Timestamp {
			if sequenceNumber <= lastEntry.ID.SequenceNumber {
				return fmt.Errorf(constants.ERR_INVALID_ID_XADD_SMALLER)
			}
		}
	}

	return nil
}

func (s *Server) XAdd(key string, entryId utils.ParsedEntryId, kvPairs map[string]string) (utils.EntryId, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastEntry, err := s.lookUpLastEntry(key)
	if err != nil {
		return utils.EntryId{}, err
	}

	ts, seq := resolveEntryId(entryId, lastEntry)

	if err := validateResolvedEntryId(ts, seq, lastEntry); err != nil {
		return utils.EntryId{}, err
	}

	updatedEntryId := utils.EntryId{
		Timestamp:      ts,
		SequenceNumber: seq,
	}

	streamVal, exists := s.store[key]
	if !exists {
		// stream doesn't exist, we create it
		newEntry := Entry{
			ID:   updatedEntryId,
			Data: kvPairs,
		}

		s.store[key] = NewStream([]Entry{newEntry})
		s.SignalAllStreamWaiters(key)
		return updatedEntryId, nil
	}

	if streamVal.Type != TypeStream {
		return utils.EntryId{}, fmt.Errorf(constants.ERR_WRONGTYPE_OPERATION)
	}

	// entry does not already exist in the stream, add it to the stream
	newEntry := Entry{
		ID:   updatedEntryId,
		Data: kvPairs,
	}

	streamVal.Entries = append(streamVal.Entries, newEntry)
	s.store[key] = streamVal
	s.SignalAllStreamWaiters(key)
	return updatedEntryId, nil
}

func (s *Server) XRange(key string, startId utils.EntryId, endId utils.EntryId) ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Lookup the key in the store
	storeValue := s.store[key]

	if storeValue.Type != TypeStream {
		return []Entry{}, fmt.Errorf(constants.ERR_WRONGTYPE_OPERATION)
	}

	var entries []Entry

	for _, entry := range storeValue.Entries {
		// current entry is smaller than start
		if entry.ID.Compare(startId) == -1 {
			continue
		}

		// current entry is larger than end
		if entry.ID.Compare(endId) == 1 {
			break
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

type XReadInput struct {
	Key           string
	EntryId       utils.EntryId
	IsLastEntryId bool
}

func (s *Server) SignalAllStreamWaiters(key string) {
	waiters := s.blXReadWaiters[key]

	// signal the channels that a value is pushed
	for _, waiter := range waiters {
		// Non blocking channel OP
		select {
		case waiter <- struct{}{}:
		default:
		}
	}

	// delete all the waiters for this key
	delete(s.blXReadWaiters, key)
}

func (s *Server) XRead(input []XReadInput, timeoutInt int) ([]Stream, error) {
	var deadline <-chan time.Time
	var isBlocking = timeoutInt >= 0

	if timeoutInt > 0 {
		deadline = time.After(time.Duration(timeoutInt) * time.Millisecond)
	}

	notifyChan := make(chan struct{}, 1)

	keys := []string{}
	for _, inputEntry := range input {
		keys = append(keys, inputEntry.Key)
	}

	// resolve all the entry ids and check which ones are passing "$" to signal the last entry id as the input
	updatedInput := []XReadInput{}
	for _, inputEntry := range input {
		s.mu.Lock()
		key := inputEntry.Key
		storeValue, exists := s.store[key]

		if exists && storeValue.Type != TypeStream {
			s.mu.Unlock()
			return []Stream{}, fmt.Errorf(constants.ERR_WRONGTYPE_OPERATION)
		}

		resolved := inputEntry // default - no "$"

		if inputEntry.IsLastEntryId { // "$" case
			if len(storeValue.Entries) == 0 {
				resolved.EntryId = utils.EntryId{}
			} else {
				resolved.EntryId = storeValue.Entries[len(storeValue.Entries)-1].ID
			}
		}

		updatedInput = append(updatedInput, resolved)
		s.mu.Unlock()
	}

	for {
		s.mu.Lock()
		s.cleanupWaiters(keys, notifyChan, s.blXReadWaiters)

		var streams []Stream = []Stream{}
		// core comparison loop
		for _, inputEntry := range updatedInput {
			key := inputEntry.Key
			entryId := inputEntry.EntryId

			storeValue := s.store[key]

			// Store value type check that it should be a stream here has already been done before the main loop, so we don't do it again
			stream := Stream{
				Key: key,
			}

			for _, entry := range storeValue.Entries {
				if entry.ID.Compare(entryId) == 1 {
					// current entry is greater than the entryId passed, this should go in to the stream
					stream.Entries = append(stream.Entries, entry)
				} else {
					continue
				}
			}

			streams = append(streams, stream)
		}

		// we wanna check if we should block or not - if the stream has ANY entries AND isBlocking is true, then we block
		var streamHasEntries bool = false
		for _, stream := range streams {
			if len(stream.Entries) > 0 {
				streamHasEntries = true
				break
			}
		}

		if streamHasEntries || !isBlocking {
			s.mu.Unlock()
			return streams, nil
		}

		// stream has no entries AND we are blocking
		for _, inputEntry := range input {
			key := inputEntry.Key
			// register a waiter
			s.blXReadWaiters[key] = append(s.blXReadWaiters[key], notifyChan)
		}

		s.mu.Unlock()

		select {
		case <-notifyChan:
			// a signal for any key
			continue
		case <-deadline:
			s.mu.Lock()
			s.cleanupWaiters(keys, notifyChan, s.blXReadWaiters)
			s.mu.Unlock()
		}

		return streams, nil
	}
}

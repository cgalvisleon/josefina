package store

import (
	"encoding/json"
	"sync"
)

/**
* counter: Numeric value guarded by its own lock. All reads and writes go
* through its methods so callers never touch the mutex directly.
**/
type counter[T ~int | ~int64 | ~uint64] struct {
	value T
	mu    sync.RWMutex
}

/**
* inc: Increments the counter by one
* @return T
**/
func (s *counter[T]) inc() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value++
	return s.value
}

/**
* dec: Decrements the counter by one
* @return T
**/
func (s *counter[T]) dec() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value--
	return s.value
}

/**
* add: Increments the counter by n
* @param n T
* @return T
**/
func (s *counter[T]) add(n T) T {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value += n
	return s.value
}

/**
* count: Returns the current value
* @return T
**/
func (s *counter[T]) count() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

/**
* set: Replaces the current value
* @param value T
**/
func (s *counter[T]) set(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
}

/**
* setMax: Replaces the current value only if value is greater
* @param value T
* @return T
**/
func (s *counter[T]) setMax(value T) T {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value > s.value {
		s.value = value
	}
	return s.value
}

/**
* MarshalJSON: Serializes the counter as its plain value
* @return []byte, error
**/
func (s *counter[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.count())
}

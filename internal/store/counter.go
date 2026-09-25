package store

import (
	"encoding/json"
	"sync"
)

/**
* counter: Valor numérico protegido por su propio lock; se usa solo a través de sus métodos.
**/
type counter[T ~int | ~int64 | ~uint64] struct {
	value T
	mu    sync.RWMutex
}

/**
* inc: Suma uno y retorna el nuevo valor.
* @return T
**/
func (s *counter[T]) inc() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value++
	return s.value
}

/**
* dec: Resta uno y retorna el nuevo valor.
* @return T
**/
func (s *counter[T]) dec() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value--
	return s.value
}

/**
* add: Suma n y retorna el nuevo valor.
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
* count: Retorna el valor actual.
* @return T
**/
func (s *counter[T]) count() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

/**
* set: Reemplaza el valor.
* @param value T
**/
func (s *counter[T]) set(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
}

/**
* setMax: Reemplaza el valor solo si el nuevo es mayor.
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
* MarshalJSON: Serializa el contador como su valor.
* @return []byte, error
**/
func (s *counter[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.count())
}

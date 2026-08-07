package jdb

import (
	"encoding/json"
	"sync"
	"time"
)

type Cache struct {
	cache map[string]*Ttl
	mu    sync.RWMutex
	store *Model
}

/**
* loadCache: Loads the cache
* @return error
**/
func (s *DB) loadCache() error {
	store, err := s.loadModel(sysSchema, "cache", 1, true)
	if err != nil {
		return err
	}

	result := &Cache{
		cache: make(map[string]*Ttl),
		store: store,
	}

	s.cache = result

	return nil
}

/**
* setCache: Stores a value in memory and in the persistent cache with an optional expiration.
* @param key string, value any, expiration time.Duration
* @return error
**/
func (s *Cache) setCache(key string, value any, expiration time.Duration) error {
	ttl, err := newTtl(value, expiration)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.cache[key] = ttl
	s.mu.Unlock()

	return s.store.put(key, ttl)
}

/**
* getCache: Returns a cached value. Checks memory first; on miss checks the persistent cache.
* Evicts the memory entry if the persistent layer reports expiration.
* @param key string, dest any
* @return bool, error
**/
func (s *Cache) getCache(key string, dest any) (bool, error) {
	s.mu.RLock()
	ttl, exists := s.cache[key]
	s.mu.RUnlock()

	if exists {
		if ttl.IsExpired() {
			s.mu.Lock()
			delete(s.cache, key)
			s.mu.Unlock()
			s.store.delete(key)
			return false, nil
		}
		if err := json.Unmarshal(ttl.Value, dest); err == nil {
			return true, nil
		}
	}

	exists, err := s.store.get(key, ttl)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if err := json.Unmarshal(ttl.Value, dest); err != nil {
		return false, err
	}

	s.mu.Lock()
	s.cache[key] = ttl
	s.mu.Unlock()
	return true, nil
}

/**
* deleteCache: Removes a value from memory and from the persistent cache.
* @param key string
* @return error
**/
func (s *Cache) deleteCache(key string) error {
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()

	return s.store.delete(key)
}

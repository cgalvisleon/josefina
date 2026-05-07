package jdb

import (
	"encoding/json"
	"time"
)

/**
* loadCache: Loads the cache
* @return error
**/
func (s *DB) loadCache() error {
	result, err := s.Define(DModel{
		Schema:  sysSchema,
		Name:    "cache",
		IsCore:  true,
		Version: 1,
	})
	if err != nil {
		return err
	}

	err = result.Init()
	if err != nil {
		return err
	}

	s.Cache = make(map[string]*Ttl)
	s.cache = result

	return nil
}

/**
* SetCache: Stores a value in memory and in the persistent cache with an optional expiration.
* @param key string, value any, expiration time.Duration
* @return error
**/
func (s *DB) SetCache(key string, value any, expiration time.Duration) error {
	ttl, err := newTtl(value, expiration)
	if err != nil {
		return err
	}

	s.muCache.Lock()
	s.Cache[key] = ttl
	s.muCache.Unlock()

	return s.cache.Put(key, ttl)
}

/**
* GetCache: Returns a cached value. Checks memory first; on miss checks the persistent cache.
* Evicts the memory entry if the persistent layer reports expiration.
* @param key string, dest any
* @return bool, error
**/
func (s *DB) GetCache(key string, dest any) (bool, error) {
	s.muCache.RLock()
	ttl, inMemory := s.Cache[key]
	s.muCache.RUnlock()

	if inMemory {
		if ttl.IsExpired() {
			s.muCache.Lock()
			delete(s.Cache, key)
			s.muCache.Unlock()
			s.cache.Remove(key)
			return false, nil
		}
		if err := json.Unmarshal(ttl.Value, dest); err == nil {
			return true, nil
		}
	}

	exists, err := s.cache.Get(key, ttl)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if err := json.Unmarshal(ttl.Value, dest); err != nil {
		return false, err
	}

	s.muCache.Lock()
	s.Cache[key] = ttl
	s.muCache.Unlock()
	return true, nil
}

/**
* DeleteCache: Removes a value from memory and from the persistent cache.
* @param key string
* @return error
**/
func (s *DB) DeleteCache(key string) error {
	s.muCache.Lock()
	delete(s.Cache, key)
	s.muCache.Unlock()

	return s.cache.Remove(key)
}

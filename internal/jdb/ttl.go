package jdb

import (
	"encoding/json"
	"time"
)

/**
* setTTL: Sets a TTL for a key
* @param idx string, value any, expiration time.Duration
* @return error
**/
func (s *Model) setTTL(idx string, value any, expiration time.Duration) error {
	if expiration == 0 {
		return nil
	}

	store, err := s.OpenStore(TTL)
	if err != nil {
		return err
	}

	ttl, err := newTtl(value, expiration)
	if err != nil {
		return err
	}

	err = store.Put(idx, ttl)
	if err != nil {
		return err
	}

	return nil
}

/**
* getTTL: Gets a TTL for a key
* @param idx string
* @return *Ttl, bool, error
**/
func (s *Model) getTTL(idx string) (*Ttl, bool, error) {
	store, err := s.OpenStore(TTL)
	if err != nil {
		return nil, false, err
	}

	var result *Ttl
	exists, err := store.Get(idx, &result)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	if result.IsExpired() {
		_, err = store.Delete(idx)
		if err != nil {
			return nil, false, err
		}
	}

	return result, true, nil
}

/**
* deleteTTL: Clears a TTL for a key
* @param idx string
* @return error
**/
func (s *Model) deleteTTL(idx string) error {
	store, err := s.OpenStore(TTL)
	if err != nil {
		return err
	}

	_, err = store.Delete(idx)
	if err != nil {
		return err
	}

	return nil
}

/**
* cleanExpired: Cleans expired TTLs
* @return error
**/
func (s *Model) cleanExpired() error {
	store, err := s.OpenStore(TTL)
	if err != nil {
		return err
	}

	if store.Count() < store.MinThresholdCompact {
		return nil
	}

	total := store.Count()
	workers := total / 1000
	if workers <= 0 {
		workers = 1
	}
	store.ForEach(func(idx string, src []byte) (bool, error) {
		var ttl *Ttl
		err := json.Unmarshal(src, &ttl)
		if err != nil {
			return true, err
		}

		if ttl.IsExpired() {
			_, err := store.Delete(idx)
			if err != nil {
				return true, err
			}
		}
		return true, nil
	}, true, 0, 0, workers)

	return nil
}

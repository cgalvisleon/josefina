package jdb

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/mem"
)

var (
	cache        *Model
	errNotExists = fmt.Errorf("NotExists")
)

/**
* initCache: Initializes the cache model
* @return error
**/
func initCache() error {
	if cache != nil {
		return nil
	}

	db, err := getDb(packageName)
	if err != nil {
		return err
	}

	cache, err = db.newModel("", "cache", true, 1)
	if err != nil {
		return err
	}
	if err := cache.init(); err != nil {
		return err
	}

	return nil
}

/**
* SetCache: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return interface{}, error
**/
func SetCache(key string, value interface{}, duration time.Duration) (*mem.Item, error) {
	result := mem.Set(key, value, duration)

	leader := node.getLeader()
	if leader != node.host && leader != "" {
		result, err := methods.setCache(leader, key, value, duration)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	err := initCache()
	if err != nil {
		return nil, err
	}

	if duration == 0 {
		err := cache.put(key, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* GetCache: Gets a cache value
* @param key string
* @return *mem.Item
**/
func GetCache(key string) (*mem.Item, bool) {
	value, err := mem.GetItem(key)
	if errors.Is(err, errNotExists) {
		leader := node.getLeader()
		if leader != node.host && leader != "" {
			result, err := methods.getCache(leader, key)
			if err != nil {
				return nil, false
			}

			return result, true
		}

		err := initCache()
		if err != nil {
			return nil, false
		}

		result := mem.Item{}
		exists, err := cache.get(key, &result)
		if err != nil {
			return nil, false
		}

		if !exists {
			return nil, false
		}

		mem.Set(key, result.Value, 0)
		return &result, true
	} else if err != nil {
		return nil, false
	}

	return value, true
}

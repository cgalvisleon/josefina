package jdb

import (
	"time"

	"github.com/cgalvisleon/et/mem"
)

var cache *Model

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
func SetCache(key string, value interface{}, duration time.Duration) (interface{}, error) {
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
* @return interface{}
**/
func GetCache(key string) interface{} {
	value, err := mem.Get(key)
	if err != nil {
		return nil
	}
	return value
}

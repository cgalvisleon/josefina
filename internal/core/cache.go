package core

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/jdb"
)

type Cache struct{}

var (
	cache        *jdb.Model
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

	db, err := jdb.GetDb(database)
	if err != nil {
		return err
	}

	cache, err = db.NewModel("", "cache", true, 1)
	if err != nil {
		return err
	}
	if err := cache.Init(); err != nil {
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
	err := initCache()
	if err != nil {
		return nil, err
	}

	if duration == 0 {
		err := cache.Put(key, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* DeleteCache: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func DeleteCache(key string) (bool, error) {
	result := mem.Delete(key)
	err := initCache()
	if err != nil {
		return false, err
	}

	err = cache.Remove(key)
	if err != nil {
		return false, err
	}

	return result, nil
}

/**
* GetCache: Gets a cache value
* @param key string
* @return *mem.Item
**/
func GetCache(key string) (*mem.Item, bool) {
	value, exists := mem.GetItem(key)
	if exists {
		return value, true
	}

	err := initCache()
	if err != nil {
		return nil, false
	}

	result := mem.Item{}
	exists, err = cache.Get(key, &result)
	if err != nil {
		return nil, false
	}

	if !exists {
		return nil, false
	}

	expiration := result.Expiration.Truncate(time.Second)
	mem.Set(key, result.Value, expiration)
	return &result, true
}

/**
* GetCacheStr: Gets a cache value as a string
* @param key string
* @return string, bool
**/
func GetCacheStr(key string) (string, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Str(), true
	}

	return "", false
}

/**
* GetCacheInt: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetCacheInt(key string) (int, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Int(), true
	}

	return 0, false
}

/**
* GetCacheInt64: Gets a cache value as an int64
* @param key string
* @return int64, bool
**/
func GetCacheInt64(key string) (int64, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Int64(), true
	}

	return 0, false
}

/**
* GetCacheFloat: Gets a cache value as a float64
* @param key string
* @return float64, bool
**/
func GetCacheFloat64(key string) (float64, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Float(), true
	}

	return 0, false
}

/**
* GetCacheBool: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetCacheBool(key string) (bool, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Bool(), true
	}

	return false, false
}

/**
* GetCacheTime: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetCacheTime(key string) (time.Time, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Time(), true
	}

	return time.Time{}, false
}

/**
* GetCacheDuration: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetCacheDuration(key string) (time.Duration, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Duration(), true
	}

	return 0, false
}

/**
* GetCacheJson: Gets a cache value as a json
* @param key string
* @return et.Json, bool
**/
func GetCacheJson(key string) (et.Json, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.Json(), true
	}

	return nil, false
}

/**
* GetCacheArrayJson: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetCacheArrayJson(key string) ([]et.Json, bool) {
	item, exists := GetCache(key)
	if exists {
		return item.ArrayJson(), true
	}

	return []et.Json{}, false
}

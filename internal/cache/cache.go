package cache

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/mod"
)

type Cache struct{}

var (
	cache    *mod.Model
	database = "josefina"
)

/**
* initModel: Initializes the cache model
* @return error
**/
func initModel() error {
	if cache != nil {
		return nil
	}

	db, err := mod.GetDb(database)
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
* Set: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return interface{}, error
**/
func Set(key string, value interface{}, duration time.Duration) (*mem.Item, error) {
	result := mem.Set(key, value, duration)
	leader, ok := getLeader()
	if ok {
		return syn.set(leader, key, value, duration)
	}

	err := initModel()
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
* Delete: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func Delete(key string) (bool, error) {
	result := mem.Delete(key)
	leader, ok := getLeader()
	if ok {
		return syn.delete(leader, key)
	}

	err := initModel()
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
* Get: Gets a cache value
* @param key string
* @return *mem.Item
**/
func Get(key string) (*mem.Item, bool) {
	value, exists := mem.GetItem(key)
	if exists {
		return value, true
	}

	leader, ok := getLeader()
	if ok {
		return syn.get(leader, key)
	}

	err := initModel()
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

	expiration := result.Expiration
	if expiration != 0 {
		expiration = result.Expiration - time.Since(result.LastUpdate)
	}
	mem.Set(key, result.Value, expiration)
	return &result, true
}

/**
* GetStr: Gets a cache value as a string
* @param key string
* @return string, bool
**/
func GetStr(key string) (string, bool) {
	item, exists := Get(key)
	if exists {
		return item.Str(), true
	}

	return "", false
}

/**
* GetInt: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetInt(key string) (int, bool) {
	item, exists := Get(key)
	if exists {
		return item.Int(), true
	}

	return 0, false
}

/**
* GetInt64: Gets a cache value as an int64
* @param key string
* @return int64, bool
**/
func GetInt64(key string) (int64, bool) {
	item, exists := Get(key)
	if exists {
		return item.Int64(), true
	}

	return 0, false
}

/**
* GetFloat: Gets a cache value as a float64
* @param key string
* @return float64, bool
**/
func GetFloat64(key string) (float64, bool) {
	item, exists := Get(key)
	if exists {
		return item.Float(), true
	}

	return 0, false
}

/**
* GetBool: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetBool(key string) (bool, bool) {
	item, exists := Get(key)
	if exists {
		return item.Bool(), true
	}

	return false, false
}

/**
* GetTime: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetTime(key string) (time.Time, bool) {
	item, exists := Get(key)
	if exists {
		return item.Time(), true
	}

	return time.Time{}, false
}

/**
* GetDuration: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetDuration(key string) (time.Duration, bool) {
	item, exists := Get(key)
	if exists {
		return item.Duration(), true
	}

	return 0, false
}

/**
* GetJson: Gets a cache value as a json
* @param key string
* @return et.Json, bool
**/
func GetJson(key string) (et.Json, bool) {
	item, exists := Get(key)
	if exists {
		return item.Json(), true
	}

	return nil, false
}

/**
* GetArrayJson: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetArrayJson(key string) ([]et.Json, bool) {
	item, exists := Get(key)
	if exists {
		return item.ArrayJson(), true
	}

	return []et.Json{}, false
}

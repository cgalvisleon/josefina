package cache

import (
	"fmt"
	"os"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/config"
)

var (
	cache *catalog.Model
)

/**
* Load: Loads the cache
* @param getLeader func() (string, bool)
* @return error
**/
func Load(getLeader func() (string, bool)) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	port := envar.GetInt("RPC_PORT", 4200)
	address := fmt.Sprintf("%s:%d", hostname, port)
	_, err = jrpc.Mount(address, syn)
	if err != nil {
		logs.Panic(err)
	}

	syn.getLeader = getLeader
	syn.address = address
	return nil
}

/**
* initModel: Initializes the cache model
* @return error
**/
func initModel() error {
	if cache != nil {
		return nil
	}

	db, err := catalog.CoreDb()
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
func set(key string, value interface{}, duration time.Duration, origin string) (*mem.Entry, error) {
	result, err := mem.Set(key, value, duration)
	if err != nil {
		return nil, err
	}

	_, imLeader := syn.getLeader()
	if !imLeader {
		logs.Debugf("Sync:%s to:%s set key:%s value:%v", syn.address, origin, key, value)
		return result, nil
	}

	err = initModel()
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
* Set: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return interface{}, error
**/
func Set(key string, value interface{}, duration time.Duration) (*mem.Entry, error) {
	result, err := set(key, value, duration, syn.address)
	if err != nil {
		return nil, err
	}

	go func() {
		nodes, err := config.GetNodes()
		if err != nil {
			return
		}
		for _, node := range nodes {
			if node == syn.address {
				continue
			}

			syn.set(node, key, value, duration, syn.address)
		}
	}()

	return result, nil
}

/**
* delete: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func delete(key, origin string) (bool, error) {
	result := mem.Delete(key)

	_, imLeader := syn.getLeader()
	if !imLeader {
		logs.Debugf("Sync:%s to:%s delete key:%s", syn.address, origin, key)
		return true, nil
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
* Delete: Deletes a cache value
* @param key string
* @return bool, error
**/
func Delete(key string) (bool, error) {
	result, err := delete(key, syn.address)
	if err != nil {
		return false, err
	}

	go func() {
		nodes, err := config.GetNodes()
		if err != nil {
			return
		}
		for _, node := range nodes {
			if node == syn.address {
				continue
			}

			syn.delete(node, key, syn.address)
		}
	}()

	return result, nil
}

/**
* Exists: Checks if a cache value exists
* @param key string
* @return bool
**/
func Exists(key string) (bool, error) {
	exists := mem.Exists(key)
	if exists {
		return true, nil
	}

	leader, imLeader := syn.getLeader()
	if !imLeader {
		return syn.exists(leader, key)
	}

	err := initModel()
	if err != nil {
		return false, err
	}

	exists, err = cache.Exists(key)
	if err != nil {
		return false, err
	}

	return exists, nil
}

/**
* Get: Gets a cache value
* @param key string
* @return *mem.Entry
**/
func Get(key string) (*mem.Entry, bool) {
	value, exists := mem.GetEntry(key)
	if exists {
		return value, true
	}

	set := func(result *mem.Entry, exists bool) (*mem.Entry, bool) {
		expiration := result.Expiration
		if expiration != 0 {
			expiration = result.Expiration - time.Since(result.LastUpdate)
		}
		mem.Set(key, result.Value, expiration)
		return result, exists
	}

	leader, imLeader := syn.getLeader()
	if !imLeader {
		result, exists := syn.get(leader, key)
		if !exists {
			return nil, false
		}
		return set(result, exists)
	}

	err := initModel()
	if err != nil {
		return nil, false
	}

	result := mem.Entry{}
	exists, err = cache.Get(key, &result)
	if err != nil {
		return nil, false
	}

	if !exists {
		return nil, false
	}

	return set(&result, exists)
}

/**
* GetStr: Gets a cache value as a string
* @param key string
* @return string, bool, error
**/
func GetStr(key string) (string, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Str()
		return result, true, err
	}

	return "", false, nil
}

/**
* GetInt: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetInt(key string) (int, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Int()
		return result, true, err
	}

	return 0, false, nil
}

/**
* GetInt64: Gets a cache value as an int64
* @param key string
* @return int64, bool
**/
func GetInt64(key string) (int64, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Int64()
		return result, true, err
	}

	return 0, false, nil
}

/**
* GetFloat: Gets a cache value as a float64
* @param key string
* @return float64, bool
**/
func GetFloat64(key string) (float64, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Float()
		return result, true, err
	}

	return 0, false, nil
}

/**
* GetBool: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetBool(key string) (bool, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Bool()
		return result, true, err
	}

	return false, false, nil
}

/**
* GetTime: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetTime(key string) (time.Time, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Time()
		return result, true, err
	}

	return time.Time{}, false, nil
}

/**
* GetDuration: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetDuration(key string) (time.Duration, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Duration()
		return result, true, err
	}

	return 0, false, nil
}

/**
* GetJson: Gets a cache value as a json
* @param key string
* @return et.Json, bool
**/
func GetJson(key string) (et.Json, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.Json()
		return result, true, err
	}

	return nil, false, nil
}

/**
* GetArrayJson: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func GetArrayJson(key string) ([]et.Json, bool, error) {
	item, exists := Get(key)
	if exists {
		result, err := item.ArrayJson()
		return result, true, err
	}

	return []et.Json{}, false, nil
}

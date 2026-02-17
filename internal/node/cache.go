package node

import (
	"time"

	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

var cache *catalog.Model

/**
* initCache: Initializes the cache model
* @return error
**/
func initCache() error {
	if cache != nil {
		return nil
	}

	db, err := node.coreDb()
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
* setCache: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return interface{}, error
**/
func setCache(key string, value interface{}, duration time.Duration) (*mem.Entry, error) {
	result, err := mem.Set(key, value, duration)
	if err != nil {
		return nil, err
	}

	err = initCache()
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
* SetCache: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return interface{}, error
**/
func (s *Node) SetCache(key string, value interface{}, duration time.Duration) error {	
	leader, imLeader := node.GetLeader()
	if imLeader {
		return s.lead.SetCache(key, value, duration)
	}

	if leader != nil {
		res := node.Request(leader, "Leader.SetCache", from)
		if res.Error != nil {
			return nil, false
		}

		var result *catalog.Model
		err := res.Get(&result)
		if err != nil {
			return nil, false
		}

		return nil
	}

	return nil
}

/**
* DeleteCache: Gets a cache value as an int
* @param key string
* @return int, bool
**/
func (s *Node) DeleteCache(key string) (bool, error) {
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
* ExistsCache: Checks if a cache value exists
* @param key string
* @return bool
**/
func (s *Node) ExistsCache(key string) (bool, error) {
	exists := mem.Exists(key)
	if exists {
		return true, nil
	}

	err := initCache()
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
* GetCache: Gets a cache value
* @param key string
* @return *mem.Entry
**/
func (s *Node) GetCache(key string) (*mem.Entry, bool) {
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

	err := initCache()
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

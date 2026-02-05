package cache

import (
	"encoding/gob"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/mem"
)

type Result struct {
	result *mem.Entry
	exists bool
}

type Cache struct {
	getLeader func() (string, bool)
	address   string
}

var (
	syn *Cache
)

func init() {
	gob.Register(mem.Entry{})
	syn = &Cache{}
}

/**
* set: Sets a cache value
* @params to string, key string, value interface{}, duration time.Duration
* @return error
**/
func (s *Cache) set(to, key string, value interface{}, duration time.Duration) (*mem.Entry, error) {
	var response *mem.Entry
	err := jrpc.CallRpc(to, "Cache.Set", et.Json{
		"key":      key,
		"value":    value,
		"duration": duration,
	}, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* Set: Sets a cache value
* @param require et.Json, response *mem.Entry
* @return error
**/
func (s *Cache) Set(require et.Json, response *mem.Entry) error {
	key := require.Str("key")
	value := require.Get("value")
	duration := time.Duration(require.Int("duration"))
	result, err := Set(key, value, duration)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* delete: Deletes a cache value
* @params to string, key string
* @return error
**/
func (s *Cache) delete(to, key string) (bool, error) {
	var response *bool
	err := jrpc.CallRpc(to, "Cache.Delete", et.Json{
		"key": key,
	}, &response)
	if err != nil {
		return false, err
	}

	return *response, nil
}

/**
* Delete: Deletes a cache value
* @param require et.Json, response *bool
* @return error
**/
func (s *Cache) Delete(require et.Json, response *bool) error {
	key := require.Str("key")
	result, err := Delete(key)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

/**
* get: Deletes a cache value
* @params to string, key string
* @return error
**/
func (s *Cache) get(to, key string) (*mem.Entry, bool) {
	var response *Result
	err := jrpc.CallRpc(to, "Cache.Delete", et.Json{
		"key": key,
	}, &response)
	if err != nil {
		return nil, false
	}

	return response.result, response.exists
}

/**
* Get: Gets a cache value
* @param require et.Json, response *Result
* @return error
**/
func (s *Cache) Get(require et.Json, response *Result) error {
	key := require.Str("key")
	result, exists := Get(key)
	if !exists {
		return errors.New("key not found")
	}

	*response = Result{
		result: result,
		exists: exists,
	}
	return nil
}

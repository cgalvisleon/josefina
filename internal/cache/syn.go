package cache

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
)

type Result struct {
	result *mem.Item
	exists bool
}

type MemCache struct{}

type getLeaderFn func() (string, bool)

var (
	syn       *MemCache
	hostname  string
	getLeader getLeaderFn
)

func init() {
	gob.Register(mem.Item{})

	hostname, _ = os.Hostname()
	port := envar.GetInt("RPC_PORT", 4200)
	hostname = fmt.Sprintf("%s:%d", hostname, port)

	syn = &MemCache{}
	_, err := jrpc.Mount(hostname, syn)
	if err != nil {
		logs.Panic(err)
	}
}

/**
* Load: Loads the cache
* @param fn getLeaderFn
* @return error
**/
func Load(fn getLeaderFn) error {
	getLeader = fn
	return nil
}

/**
* set: Sets a cache value
* @params to string, key string, value interface{}, duration time.Duration
* @return error
**/
func (s *MemCache) set(to, key string, value interface{}, duration time.Duration) (*mem.Item, error) {
	var response *mem.Item
	err := jrpc.CallRpc(to, "MemCache.Set", et.Json{
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
* @param require et.Json, response *mem.Item
* @return error
**/
func (s *MemCache) Set(require et.Json, response *mem.Item) error {
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
func (s *MemCache) delete(to, key string) (bool, error) {
	var response *bool
	err := jrpc.CallRpc(to, "MemCache.Delete", et.Json{
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
func (s *MemCache) Delete(require et.Json, response *bool) error {
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
func (s *MemCache) get(to, key string) (*mem.Item, bool) {
	var response *Result
	err := jrpc.CallRpc(to, "MemCache.Delete", et.Json{
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
func (s *MemCache) Get(require et.Json, response *Result) error {
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

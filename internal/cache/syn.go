package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
)

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
* @param getLeader getLeaderFn
* @return error
**/
func Load(getLeader getLeaderFn) error {
	getLeader = getLeader
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

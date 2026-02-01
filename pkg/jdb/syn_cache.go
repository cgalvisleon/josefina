package jdb

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* setCache
* @param to, key string, value interface{}, duration time.Duration
* @return error
**/
func (s *Syn) setCache(to, key string, value interface{}, duration time.Duration) (*mem.Item, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"key":      key,
		"value":    value,
		"duration": duration,
	}
	var reply *mem.Item
	err := jrpc.CallRpc(to, "Syn.SetCache", data, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* SetCache
* @param require et.Json, response *mem.Item
* @return error
**/
func (s *Syn) SetCache(require et.Json, response *mem.Item) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	key := require.Str("key")
	value := require.Str("value")
	duration := time.Duration(require.Int("duration"))
	result, err := SetCache(key, value, duration)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* getCache
* @param to, key string
* @return error
**/
func (s *Syn) getCache(to, key string) (*mem.Item, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var reply *mem.Item
	err := jrpc.CallRpc(to, "Syn.GetCache", key, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* GetCache
* @param require string, response *mem.Item
* @return error
**/
func (s *Syn) GetCache(require string, response *mem.Item) bool {
	if node == nil {
		return false
	}

	result, exists := GetCache(require)
	response = result
	return exists
}

/**
* deleteCache
* @param to, key string
* @return error
**/
func (s *Syn) deleteCache(to, key string) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var reply bool
	err := jrpc.CallRpc(to, "Syn.DeleteCache", key, &reply)
	if err != nil {
		return false, err
	}

	return reply, nil
}

/**
* DeleteCache
* @param require string, response *mem.Item
* @return error
**/
func (s *Syn) DeleteCache(require string, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	result, err := DeleteCache(require)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

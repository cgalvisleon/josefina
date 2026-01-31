package jdb

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Persist struct{}

var persist *Persist

func init() {
	persist = &Persist{}
}

/**
* put
* @param to, key string
* @return error
**/
func (s *Persist) put(from *From, idx string, data any) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
		"data": data,
	}
	var reply bool
	err := jrpc.CallRpc(from.Host, "Methods.Put", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* Put
* @param require et.Json, response *bool
* @return error
**/
func (s *Persist) Put(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	data := require.Get("data")
	err := put(from, idx, data)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* remove
* @param to, key string
* @return error
**/
func (s *Persist) remove(from *From, idx string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var reply bool
	err := jrpc.CallRpc(from.Host, "Methods.Remove", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* Remove
* @param require et.Json, response *bool
* @return error
**/
func (s *Persist) Remove(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	err := remove(from, idx)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* get
* @param to, idx string, dest any
* @return error
**/
func (s *Persist) get(from *From, idx string, dest any) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var reply AnyResult
	err := jrpc.CallRpc(from.Host, "Methods.Get", args, &reply)
	if err != nil {
		return false, err
	}

	dest = reply.Dest
	return reply.Ok, nil
}

/**
* Get
* @param require et.Json, response *AnyResult
* @return error
**/
func (s *Persist) Get(require et.Json, response *AnyResult) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	var dest any
	ok, err := get(from, idx, &dest)
	if err != nil {
		return err
	}

	*response = AnyResult{
		Dest: dest,
		Ok:   ok,
	}
	return nil
}

/**
* putObject
* @param to, idx string, dest any
* @return error
**/
func (s *Persist) putObject(from *From, idx string, dest et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	err := jrpc.CallRpc(from.Host, "Methods.PutObject", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* PutObject
* @param require et.Json, response et.Json
* @return error
**/
func (s *Persist) PutObject(require et.Json, response et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	var dest et.Json
	err := putObject(from, idx, dest)
	if err != nil {
		return err
	}

	response = dest
	return nil
}

/**
* removeObject
* @param to, idx string, dest any
* @return error
**/
func (s *Persist) removeObject(from *From, idx string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var dest bool
	err := jrpc.CallRpc(from.Host, "Methods.RemoveObject", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* RemoveObject
* @param require et.Json, response *bool
* @return error
**/
func (s *Persist) RemoveObject(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	err := removeObject(from, idx)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* isExisted
* @param to, idx string, dest any
* @return error
**/
func (s *Persist) isExisted(from *From, field, idx string) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from":  from,
		"field": field,
		"idx":   idx,
	}
	var dest bool
	err := jrpc.CallRpc(from.Host, "Methods.IsExisted", args, &dest)
	if err != nil {
		return false, err
	}

	return dest, nil
}

/**
* IsExisted
* @param require et.Json, response *bool
* @return error
**/
func (s *Persist) IsExisted(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	field := require.Str("field")
	idx := require.Str("idx")
	existed, err := isExisted(from, field, idx)
	if err != nil {
		return err
	}

	*response = existed
	return nil
}

/**
* count
* @param to, idx string, dest any
* @return error
**/
func (s *Persist) count(from *From) (int, error) {
	if node == nil {
		return 0, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var dest int
	err := jrpc.CallRpc(from.Host, "Methods.Count", from, &dest)
	if err != nil {
		return 0, err
	}

	return dest, nil
}

/**
* Count
* @param require *From, response *int
* @return error
**/
func (s *Persist) Count(require *From, response *int) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	existed, err := count(require)
	if err != nil {
		return err
	}

	*response = existed
	return nil
}

/**
* put: Puts an object into the model
* @param from *From, key string, data any
* @return error
**/
func put(from *From, idx string, data any) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return persist.put(from, idx, data)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.put(idx, data)
}

/**
* remove: Removes an object from the model
* @param from *From, idx string
* @return error
**/
func remove(from *From, idx string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return persist.remove(from, idx)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.remove(idx)
}

/**
* get: Gets an object from the model
* @param from *From, idx string, dest any
* @return bool, error
**/
func get(from *From, idx string, dest any) (bool, error) {
	if !node.started {
		return false, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return persist.get(from, idx, dest)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return false, fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.get(idx, dest)
}

/**
* putObject: Puts an object into the model
* @param model *Model, idx string, data et.Json
* @return error
**/
func putObject(from *From, idx string, data et.Json) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return persist.putObject(from, idx, data)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.putObject(idx, data)
}

/**
* getObject: Gets the model as object
* @param idx string
* @return et.Json, error
**/
func getObject(from *From, idx string, dest et.Json) (bool, error) {
	return get(from, idx, &dest)
}

/**
* removeObject: Removes an object from the model
* @param model *Model, key string
* @return error
**/
func removeObject(from *From, idx string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return persist.removeObject(from, idx)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.removeObject(key)
}

/**
* isExisted
* @param from *From, field string, key string
* @return (bool, error)
**/
func isExisted(from *From, field, idx string) (bool, error) {
	if !node.started {
		return false, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return persist.isExisted(from, field, idx)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return false, fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.isExisted(field, idx)
}

/**
* count
* @param from *From
* @return (int, error)
**/
func count(from *From) (int, error) {
	if !node.started {
		return 0, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return persist.count(from)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return 0, fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.count()
}

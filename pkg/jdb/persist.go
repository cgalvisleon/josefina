package jdb

import (
	"errors"

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
	err := from.Put(idx, data)
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

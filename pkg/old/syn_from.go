package old

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* put: Puts a value
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) put(from *From, idx string, data any) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
		"data": data,
	}
	var reply bool
	err := jrpc.CallRpc(from.Host, "Syn.Put", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* Put: Puts a value
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) Put(require et.Json, response *bool) error {
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
* remove: Removes a value
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) remove(from *From, idx string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var reply bool
	err := jrpc.CallRpc(from.Host, "Syn.Remove", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* Remove: Removes a value
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) Remove(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	err := from.Remove(idx)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* get: Gets a value
* @param require et.Json, response *AnyResult
* @return error
**/
func (s *Syn) get(from *From, idx string, dest any) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var reply AnyResult
	err := jrpc.CallRpc(from.Host, "Syn.Get", args, &reply)
	if err != nil {
		return false, err
	}

	dest = reply.Dest
	return reply.Ok, nil
}

/**
* Get: Gets a value
* @param require et.Json, response *AnyResult
* @return error
**/
func (s *Syn) Get(require et.Json, response *AnyResult) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	var dest any
	ok, err := from.Get(idx, &dest)
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
* putObject: Puts an object
* @param require et.Json, response et.Json
* @return error
**/
func (s *Syn) putObject(from *From, idx string, dest et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	err := jrpc.CallRpc(from.Host, "Syn.PutObject", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* PutObject: Puts an object
* @param require et.Json, response et.Json
* @return error
**/
func (s *Syn) PutObject(require et.Json, response et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	var dest et.Json
	err := from.PutObject(idx, dest)
	if err != nil {
		return err
	}

	response = dest
	return nil
}

/**
* removeObject: Removes an object
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) removeObject(from *From, idx string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var dest bool
	err := jrpc.CallRpc(from.Host, "Syn.RemoveObject", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* RemoveObject: Removes an object
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) RemoveObject(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	err := from.RemoveObject(idx)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* isExisted: Checks if an object exists
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) isExisted(from *From, field, idx string) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from":  from,
		"field": field,
		"idx":   idx,
	}
	var dest bool
	err := jrpc.CallRpc(from.Host, "Syn.IsExisted", args, &dest)
	if err != nil {
		return false, err
	}

	return dest, nil
}

/**
* IsExisted: Checks if an object exists
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) IsExisted(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	field := require.Str("field")
	idx := require.Str("idx")
	existed, err := from.IsExisted(field, idx)
	if err != nil {
		return err
	}

	*response = existed
	return nil
}

/**
* count: Counts the number of objects
* @param require *From, response *int
* @return error
**/
func (s *Syn) count(from *From) (int, error) {
	if node == nil {
		return 0, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var dest int
	err := jrpc.CallRpc(from.Host, "Syn.Count", from, &dest)
	if err != nil {
		return 0, err
	}

	return dest, nil
}

/**
* Count: Counts the number of objects
* @param require *From, response *int
* @return error
**/
func (s *Syn) Count(require *From, response *int) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	existed, err := require.Count()
	if err != nil {
		return err
	}

	*response = existed
	return nil
}

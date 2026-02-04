package mod

import (
	"encoding/gob"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type DbResult struct {
	Exists bool
	Db     *DB
}

type ModelResult struct {
	Exists bool
	Model  *Model
}

type Mod struct {
	getLeader func() (string, bool)
	address   string
}

var (
	syn *Mod
)

func init() {
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(DB{})
	gob.Register(Schema{})
	gob.Register(Model{})
	gob.Register(Tx{})
	gob.Register(Transaction{})
	gob.Register(DbResult{})
	gob.Register(ModelResult{})
	syn = &Mod{}
}

/**
* getModel
* @params from *From
* @return (*Model, bool)
**/
func (s *Mod) getModel(from *From) (*Model, bool) {
	leader, ok := s.getLeader()
	if !ok {
		return nil, false
	}

	var response *ModelResult
	err := jrpc.CallRpc(leader, "Node.GetModel", from, &response)
	if err != nil {
		return nil, false
	}

	return response.Model, response.Exists
}

/**
* LoadModel: Loads a model
* @param require *Model, response *Model
* @return error
**/
func (s *Mod) LoadModel(require *Model, response *Model) error {
	result, err := loadModel(require)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* removeObject
* @params from *From, idx string
* @return error
**/
func (s *Mod) removeObject(from *From, idx string) error {
	model, exists := s.getModel(from)
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	var response bool
	err := jrpc.CallRpc(model.Address, "Mod.RemoveObject", et.Json{
		"from": from,
		"idx":  idx,
	}, &response)
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
func (s *Mod) RemoveObject(require et.Json, response *bool) error {
	from := ToFrom(require.Json("from"))
	idx := require.Str("idx")
	model, exists := GetModel(from)
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	err := model.RemoveObject(idx)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* putObject
* @params from *From, idx string, data et.Json
* @return error
**/
func (s *Mod) putObject(from *From, idx string, data et.Json) error {
	model, exists := s.getModel(from)
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	var response bool
	err := jrpc.CallRpc(model.Address, "Mod.PutObject", et.Json{
		"from": from,
		"idx":  idx,
		"data": data,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* PutObject: Puts an object
* @param require et.Json, response *bool
* @return error
**/
func (s *Mod) PutObject(require et.Json, response *bool) error {
	from := ToFrom(require.Json("from"))
	idx := require.Str("idx")
	data := require.Json("data")
	model, exists := GetModel(from)
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	err := model.PutObject(idx, data)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* isExisted
* @params from *From, field, idx string
* @return error
**/
func (s *Mod) isExisted(from *From, field, idx string) (bool, error) {
	model, exists := s.getModel(from)
	if !exists {
		return false, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	var response bool
	err := jrpc.CallRpc(model.Address, "Mod.IsExisted", et.Json{
		"from":  from,
		"field": field,
		"idx":   idx,
	}, &response)
	if err != nil {
		return false, err
	}

	return response, nil
}

/**
* IsExisted: Checks if an object exists
* @param require et.Json, response *bool
* @return error
**/
func (s *Mod) IsExisteds(require et.Json, response *bool) error {
	from := ToFrom(require.Json("from"))
	field := require.Str("field")
	idx := require.Str("idx")
	model, exists := GetModel(from)
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	exists, err := model.IsExisted(field, idx)
	if err != nil {
		return err
	}

	*response = exists
	return nil
}

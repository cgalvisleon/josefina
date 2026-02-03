package mod

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Mod struct{}

var (
	syn     *Mod
	address string
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

	hostname, _ := os.Hostname()
	port := envar.GetInt("RPC_PORT", 4200)
	address = fmt.Sprintf("%s:%d", hostname, port)

	syn = &Mod{}
	_, err := jrpc.Mount(address, syn)
	if err != nil {
		logs.Panic(err)
	}
}

/**
* removeObject
* @params from *From, idx string
* @return error
**/
func (s *Mod) removeObject(from *From, idx string) error {
	var response bool
	err := jrpc.CallRpc(from.Address, "Mod.RemoveObject", et.Json{
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
	var response bool
	err := jrpc.CallRpc(from.Address, "Mod.PutObject", et.Json{
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
	var response bool
	err := jrpc.CallRpc(from.Address, "Mod.IsExisted", et.Json{
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

/**
* IsExisted: Checks if an object exists
* @param require et.Json, response *bool
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

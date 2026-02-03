package dbs

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
)

type Dbs struct{}

var (
	syn     *Dbs
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

	syn = &Dbs{}
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
func (s *Dbs) removeObject(from *From, idx string) error {
	var response bool
	err := jrpc.CallRpc(from.Address, "Dbs.RemoveObject", et.Json{
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
func (s *Dbs) RemoveObject(require et.Json, response *bool) error {
	from := ToFrom(require.Json("from"))
	idx := require.Str("idx")
	model, err := getModel(from)
	if err != nil {
		return err
	}
	err = model.RemoveObject(idx)
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
func (s *Dbs) putObject(from *From, idx string, data et.Json) error {
	var response bool
	err := jrpc.CallRpc(from.Address, "Dbs.PutObject", et.Json{
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
func (s *Dbs) PutObject(require et.Json, response *bool) error {
	from := ToFrom(require.Json("from"))
	idx := require.Str("idx")
	data := require.Json("data")
	model, err := getModel(from)
	if err != nil {
		return err
	}
	err = model.PutObject(idx, data)
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
func (s *Dbs) isExisted(from *From, field, idx string) (bool, error) {
	var response bool
	err := jrpc.CallRpc(from.Address, "Dbs.IsExisted", et.Json{
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
func (s *Dbs) IsExisted(require et.Json, response *bool) error {
	from := ToFrom(require.Json("from"))
	field := require.Str("field")
	idx := require.Str("idx")
	model, err := getModel(from)
	if err != nil {
		return err
	}
	exists, err := model.IsExisted(field, idx)
	if err != nil {
		return err
	}

	*response = exists
	return nil
}

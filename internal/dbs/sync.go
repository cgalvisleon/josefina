package dbs

import (
	"encoding/gob"
	"net/rpc"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
)

type Sync struct{}

var sync *Sync

func init() {
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(&DB{})
	gob.Register(&Schema{})
	gob.Register(&Model{})
	gob.Register(&Tx{})
	gob.Register(&Transaction{})

	sync = &Sync{}
	err := rpc.Register(sync)
	if err != nil {
		logs.Panic(err)
	}
}

/**
* RemoveObject: Removes an object
* @param require et.Json, response *bool
* @return error
**/
func (s *Sync) RemoveObject(require et.Json, response *bool) error {
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
* PutObject: Puts an object
* @param require et.Json, response *bool
* @return error
**/
func (s *Sync) PutObject(require et.Json, response *bool) error {
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
* removeObject
* @params from *From, idx string
* @return error
**/
func removeObject(from *From, idx string) error {
	var response bool
	err := jrpc.CallRpc(from.Host, "Sync.RemoveObject", et.Json{
		"from": from,
		"idx":  idx,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* putObject
* @params from *From, idx string, data et.Json
* @return error
**/
func putObject(from *From, idx string, data et.Json) error {
	var response bool
	err := jrpc.CallRpc(from.Host, "Sync.PutObject", et.Json{
		"from": from,
		"idx":  idx,
		"data": data,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

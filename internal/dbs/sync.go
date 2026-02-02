package dbs

import (
	"net/rpc"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Sync struct{}

var sync *Sync

func init() {
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

func removeObject(from *From, idx string) error {
	return nil
}

func putObject(from *From, idx string, data et.Json) error {
	return nil
}

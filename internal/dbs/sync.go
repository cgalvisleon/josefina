package dbs

import (
	"net/rpc"

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

func (s *Sync) GetDb(name string) (*DB, error) {
	return nil, nil
}

func removeObject(fromidx string) error {
	return nil
}

func putObject(idx string, data et.Json) error {
	return nil
}

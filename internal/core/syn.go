package core

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/internal/mod"
)

type Core struct {
	getLeader func() (string, bool)
	address   string
}

var (
	syn *Core
)

func init() {
	syn = &Core{}
}

/**
* createDb: Creates a database
* @params to string, name string
* @return *mod.DB, error
**/
func (s *Core) createDb(to, name string) (*mod.DB, error) {
	var response *mod.DB
	err := jrpc.CallRpc(to, "Core.CreateDb", et.Json{
		"name": name,
	}, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* CreateDb: Creates a database
* @param require et.Json, response *mod.DB
* @return error
**/
func (s *Core) CreateDb(require et.Json, response *mod.DB) error {
	name := require.Str("name")
	result, err := CreateDb(name)
	if err != nil {
		return err
	}

	response = result
	return nil
}

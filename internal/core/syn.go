package core

import (
	"encoding/gob"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/internal/mod"
)

type Core struct {
	getLeader func() (string, bool)
	address   string
}

type DbResult struct {
	Exists bool
	Db     *mod.DB
}

var (
	syn *Core
)

func init() {
	gob.Register(DbResult{})
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

/**
* getDb: Gets a database
* @params to, name string, dest *DbResult
* @return bool, error
**/
func (s *Core) getDb(to, name string, dest *mod.DB) (bool, error) {
	var response *DbResult
	err := jrpc.CallRpc(to, "Core.GetDb", et.Json{
		"name": name,
	}, &response)
	if err != nil {
		return false, err
	}

	dest = response.Db
	return response.Exists, nil
}

/**
* GetDb: Gets a database
* @param require et.Json, response *mod.DB
* @return error
**/
func (s *Core) GetDb(require et.Json, response *DbResult) error {
	name := require.Str("name")
	exists, err := GetDb(name, response.Db)
	if err != nil {
		return err
	}

	response.Exists = exists
	return nil
}

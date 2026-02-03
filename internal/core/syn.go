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

type ModelResult struct {
	Exists bool
	Model  *mod.Model
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

/**
* dropDb: Drops a database
* @params to, name string
* @return error
**/
func (s *Core) dropDb(to, name string) error {
	var response *DbResult
	err := jrpc.CallRpc(to, "Core.DropDb", et.Json{
		"name": name,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* DropDb: Drops a database
* @param require et.Json, response *bool
* @return error
**/
func (s *Core) DropDb(require et.Json, response *bool) error {
	name := require.Str("name")
	err := DropDb(name)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* createModel: Creates a model
* @params to, database, schema, name string, version int
* @return *mod.Model, error
**/
func (s *Core) createModel(to, database, schema, name string, version int) (*mod.Model, error) {
	var response *mod.Model
	err := jrpc.CallRpc(to, "Core.CreateModel", et.Json{
		"database": database,
		"schema":   schema,
		"name":     name,
		"version":  version,
	}, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* CreateModel: Creates a model
* @param require et.Json, response *mod.Model
* @return error
**/
func (s *Core) CreateModel(require et.Json, response *mod.Model) error {
	database := require.Str("database")
	schema := require.Str("schema")
	name := require.Str("name")
	version := require.Int("version")
	result, err := CreateModel(database, schema, name, version)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* getModel: Gets a model
* @params to string, from *mod.From, dest *mod.Model
* @return bool, error
**/
func (s *Core) getModel(to string, from *mod.From, dest *mod.Model) (bool, error) {
	var response *ModelResult
	err := jrpc.CallRpc(to, "Core.GetModel", et.Json{
		"database": from.Database,
		"schema":   from.Schema,
		"name":     from.Name,
	}, &response)
	if err != nil {
		return false, err
	}

	dest = response.Model
	return response.Exists, nil
}

/**
* GetModel: Gets a model
* @param require et.Json, response *ModelResult
* @return error
**/
func (s *Core) GetModel(require et.Json, response *ModelResult) error {
	from := &mod.From{
		Database: require.Str("database"),
		Schema:   require.Str("schema"),
		Name:     require.Str("name"),
	}
	exists, err := GetModel(from, response.Model)
	if err != nil {
		return err
	}

	response.Exists = exists
	return nil
}

/**
* dropModel: Drops a model
* @params to string, from *mod.From
* @return error
**/
func (s *Core) dropModel(to string, from *mod.From) error {
	var response *ModelResult
	err := jrpc.CallRpc(to, "Core.DropModel", et.Json{
		"database": from.Database,
		"schema":   from.Schema,
		"name":     from.Name,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* DropModel: Gets a model
* @param require et.Json, response *bool
* @return error
**/
func (s *Core) DropModel(require et.Json, response *bool) error {
	from := &mod.From{
		Database: require.Str("database"),
		Schema:   require.Str("schema"),
		Name:     require.Str("name"),
	}
	err := DropModel(from)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* createSerie: Creates a serie
* @params to string, tag, format string, value int
* @return error
**/
func (s *Core) createSerie(to, tag, format string, value int) error {
	var response *ModelResult
	err := jrpc.CallRpc(to, "Core.CreateSerie", et.Json{
		"tag":    tag,
		"format": format,
		"value":  value,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* CreateSerie: Gets a model
* @param require et.Json, response *bool
* @return error
**/
func (s *Core) CreateSerie(require et.Json, response *bool) error {
	tag := require.Str("tag")
	format := require.Str("format")
	value := require.Int("value")
	err := CreateSerie(tag, format, value)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* setSerie: Creates a serie
* @params to string, tag, format string, value int
* @return error
**/
func (s *Core) setSerie(to, tag string, value int) error {
	var response *ModelResult
	err := jrpc.CallRpc(to, "Core.SetSerie", et.Json{
		"tag":   tag,
		"value": value,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetSerie: Gets a model
* @param require et.Json, response *bool
* @return error
**/
func (s *Core) SetSerie(require et.Json, response *bool) error {
	tag := require.Str("tag")
	value := require.Int("value")
	err := SetSerie(tag, value)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

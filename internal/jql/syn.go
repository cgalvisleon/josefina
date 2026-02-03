package jql

import (
	"encoding/gob"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/internal/dbs"
)

type getLeaderFn func() (string, bool)

type Jql struct{}

var (
	syn *Jql
)

func init() {
	gob.Register(Ql{})
	gob.Register(Cmd{})
}

/**
* exec: Executes a comand
* @params to string, query et.Json
* @return error
**/
func (s *Jql) exec(cmd Cmd) (et.Items, error) {
	var response et.Items
	err := jrpc.CallRpc(cmd.address, "Jql.Exec", cmd, &response)
	if err != nil {
		return et.Items{}, err
	}

	return response, nil
}

/**
* Exec: Executes a comand
* @param require et.Json, response *et.Items
* @return error
**/
func (s *Jql) Exec(require Cmd, response et.Items) error {
	response = et.Items{}
	return nil
}

/**
* run: Executes a query
* @params to string, query et.Json
* @return error
**/
func (s *Jql) run(ql Ql) (et.Items, error) {
	var response et.Items
	err := jrpc.CallRpc(ql.address, "Jql.Run", ql, &response)
	if err != nil {
		return et.Items{}, err
	}

	return response, nil
}

/**
* Run: Executes a query
* @param require et.Json, response *et.Items
* @return error
**/
func (s *Jql) Run(require Ql, response et.Items) error {
	response = et.Items{}
	return nil
}

/**
* getDb: Gets a database
* @param to string, name string
* @return *DB, error
**/
func (s *Jql) getDb(to string, name string) (*dbs.DB, error) {
	var response *dbs.DB
	err := jrpc.CallRpc(to, "Nodes.GetDb", name, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* GetDb: Gets a database
* @param require string, response *DB
* @return error
**/
func (s *Jql) GetDb(require string, response *dbs.DB) error {
	db, err := node.getDb(require)
	if err != nil {
		return err
	}

	*response = *db
	return nil
}

/**
* setDb: Saves a database
* @param to string, db *DB
* @return error
**/
func (s *Jql) setDb(to string, db *dbs.DB) error {
	var response bool
	err := jrpc.CallRpc(to, "Nodes.SetDb", db, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetDb: Saves a database
* @param model *DB
* @return bool, error
**/
func (s *Jql) SetDb(require *dbs.DB, response *bool) error {
	err := node.setDb(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* getModel: Gets a model
* @param to string, database, schema, model string
* @return *Model, error
**/
func (s *Jql) getModel(to, database, schema, name string) (*dbs.Model, error) {
	var response *dbs.Model
	err := jrpc.CallRpc(to, "Nodes.GetModel", et.Json{
		"database": database,
		"schema":   schema,
		"name":     name,
	}, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* GetModel: Gets a model
* @param require et.Json, response *Model
* @return error
**/
func (s *Jql) GetModel(require et.Json, response *dbs.Model) error {
	database := require.Str("database")
	schema := require.Str("schema")
	name := require.Str("name")
	result, err := node.getModel(database, schema, name)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* loadModel: Loads a model
* @param to string, model *Model
* @return error
**/
func (s *Jql) loadModel(to string, model *dbs.Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Nodes.LoadModel", model, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* LoadModel: Loads a model
* @param require *Model, response true
* @return error
**/
func (s *Jql) LoadModel(require *dbs.Model, response *bool) error {
	err := node.loadModel(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* setModel: Saves a model
* @param to string, model *Model
* @return error
**/
func (s *Jql) setModel(to string, model *dbs.Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Nodes.SetModel", model, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetModel: Saves a model
* @param model *Model
* @return bool, error
**/
func (s *Jql) SetModel(require *dbs.Model, response *bool) error {
	err := node.setModel(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

package rds

import (
	"github.com/cgalvisleon/et/et"
)

type Follow struct{}

var follow *Follow

/**
* getDB
* @param name string
* @return *DB, error
**/
func (s *Follow) getDB(name string) (*DB, error) {
	var response DB
	err := callRpc(node.master, "Master.GetDB", name, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

/**
* getModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Follow) getModel(database, schema, model string) (*Model, error) {
	var response Model
	err := callRpc(node.master, "Master.GetModel", et.Json{
		"database": database,
		"schema":   schema,
		"model":    model,
	}, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

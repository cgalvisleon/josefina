package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Master struct{}

var master *Master

/**
* Ping: Pings the master
* @param response *string
* @return error
**/
func (s *Master) Ping(require string, response *string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	node.addNode(require)
	logs.Log(packageName, "ping:", require)
	*response = "pong"
	return nil
}

/**
* GetDB: Returns a database by name
* @param name string
* @return *DB, error
**/
func (s *Master) GetDB(require string, response *DB) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	result, err := node.getDb(require)
	if err != nil {
		return err
	}

	*response = *result
	return nil
}

/**
* GetDB: Returns a database by name
* @param name string
* @return *DB, error
**/
func (s *Master) GetModel(require et.Json, response *Model) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	database := require.Str("database")
	schema := require.Str("schema")
	model := require.Str("model")
	result, err := node.getModel(database, schema, model)
	if err != nil {
		return err
	}

	*response = *result
	return nil
}

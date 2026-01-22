package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Methods struct{}

var methods *Methods

/**
* ping
* @return error
**/
func (s *Methods) ping() error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var response string
	err := jrpc.CallRpc(node.master, "Methods.Ping", node.host, &response)
	if err != nil {
		return err
	}

	logs.Logf(packageName, "%s:%s", response, node.master)
	return nil
}

/**
* Ping: Pings the master
* @param response *string
* @return error
**/
func (s *Methods) Ping(require string, response *string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	logs.Log(packageName, "ping:", require)
	*response = "pong"
	return nil
}

/**
* getVote
* @param tag, host string
* @return string, error
**/
func (s *Methods) getVote(tag, host string) (string, error) {
	data := et.Json{
		"tag":  tag,
		"host": host,
	}
	var response string
	err := jrpc.CallRpc(node.master, "Methods.GetVote", data, &response)
	if err != nil {
		return "", err
	}

	return response, nil
}

/**
* GetVote: Returns the votes for a tag
* @param require et.Json, response *string
* @return error
**/
func (s *Methods) GetVote(require et.Json, response *string) error {
	tag := require.Str("tag")
	host := require.Str("host")
	result, err := getVote(tag, host)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

/**
* getDB
* @param name string
* @return *DB, error
**/
func (s *Methods) getDB(name string) (*DB, error) {
	var response DB
	err := jrpc.CallRpc(node.master, "Methods.GetDB", name, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

/**
* GetDB: Returns a database by name
* @param require string, response *DB
* @return error
**/
func (s *Methods) GetDB(require string, response *DB) error {
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
* getModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Methods) getModel(database, schema, model string) (*Model, error) {
	var response Model
	err := jrpc.CallRpc(node.master, "Methods.GetModel", et.Json{
		"database": database,
		"schema":   schema,
		"model":    model,
	}, &response)
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
func (s *Methods) GetModel(require et.Json, response *Model) error {
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

/**
* SignIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func SignIn(device, database, username, password string) (*Session, error) {
	if node == nil {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	return signIn(device, database, username, password)
}

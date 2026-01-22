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
func (s *Methods) ping(to string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var response string
	err := jrpc.CallRpc(to, "Methods.Ping", node.host, &response)
	if err != nil {
		return err
	}

	logs.Logf(packageName, "%s:%s", response, to)
	return nil
}

/**
* Ping: Pings the leader
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
* vote
* @param tag, host string
* @return error
**/
func (s *Methods) vote(to, tag, host string) error {
	data := et.Json{
		"tag":  tag,
		"host": host,
	}
	var response string
	err := jrpc.CallRpc(to, "Methods.Vote", data, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* GetVote: Returns the votes for a tag
* @param require et.Json, response *string
* @return error
**/
func (s *Methods) Vote(require et.Json, response *string) error {
	tag := require.Str("tag")
	host := require.Str("host")
	go vote(tag, host)

	return nil
}

/**
* vote
* @param tag, host string
* @return error
**/
func (s *Methods) getVote(to, tag string) (string, error) {
	var response string
	err := jrpc.CallRpc(to, "Methods.GetVote", tag, &response)
	if err != nil {
		return "", err
	}

	return response, nil
}

/**
* GetVote: Returns the votes for a tag
* @param require string, response *string
* @return error
**/
func (s *Methods) GetVote(require string, response *string) error {
	tag := require
	result := make(chan string)
	go func() {
		res := getVote(tag)
		result <- res
	}()

	*response = <-result
	return nil
}

/**
* getDB
* @param name string
* @return *DB, error
**/
func (s *Methods) getDB(name string) (*DB, error) {
	var response DB
	err := jrpc.CallRpc(node.leader, "Methods.GetDB", name, &response)
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
func (s *Methods) getModel(database, schema, model, host string) (*Model, error) {
	var response Model
	err := jrpc.CallRpc(node.leader, "Methods.GetModel", et.Json{
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
	host := require.Str("host")
	result, err := node.getModel(database, schema, model, host)
	if err != nil {
		return err
	}

	*response = *result
	return nil
}

/**
* signIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func (s *Methods) signIn(device, database, username, password string) (*Session, error) {
	var response Session
	err := jrpc.CallRpc(node.leader, "Methods.SignIn", et.Json{
		"device":   device,
		"database": database,
		"username": username,
		"password": password,
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
func (s *Methods) SignIn(require et.Json, response *Session) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	device := require.Str("device")
	database := require.Str("database")
	username := require.Str("username")
	password := require.Str("password")
	result, err := node.signIn(device, database, username, password)
	if err != nil {
		return err
	}

	*response = *result
	return nil
}

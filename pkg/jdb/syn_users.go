package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* createUser: Creates a user
* @param to, username, password string
* @return error
**/
func (s *Syn) createUser(to, username, password string) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.CreateUser", et.Json{
		"username": username,
		"password": password,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* CreateUser
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) CreateUser(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	password := require.Str("password")
	err := createUser(username, password)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* dropUser: Drops a user
* @param to, username string
* @return error
**/
func (s *Syn) dropUser(to, username string) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.DropUser", et.Json{
		"username": username,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* DropUser: Drops a user
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) DropUser(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	err := dropUser(username)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* changuePassword: Changes a user's password
* @param device, username, password string
* @return *Session, error
**/
func (s *Syn) changuePassword(to, username, password string) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.ChanguePassword", et.Json{
		"username": username,
		"password": password,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* ChanguePassword: Changes a user's password
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) ChanguePassword(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	password := require.Str("password")
	err := changuePassword(username, password)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* auth: Authenticates a user
* @param device, username, password string
* @return *Session, error
**/
func (s *Syn) auth(to, device, database, username, password string) (*Session, error) {
	var response Session
	err := jrpc.CallRpc(to, "Syn.Auth", et.Json{
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
* Auth: Authenticates a user
* @param require et.Json, response *Session
* @return error
**/
func (s *Syn) Auth(require et.Json, response *Session) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	device := require.Str("device")
	database := require.Str("database")
	username := require.Str("username")
	password := require.Str("password")
	result, err := Auth(device, database, username, password)
	if err != nil {
		return err
	}

	*response = *result
	return nil
}

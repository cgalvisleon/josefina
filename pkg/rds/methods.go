package rds

import (
	"fmt"

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

	address := fmt.Sprintf(`%s:%d`, node.Host, node.Port)
	var response string
	err := callRpc(node.master, "Master.Ping", address, &response)
	if err != nil {
		return err
	}

	logs.Logf(packageName, "%s:%s", response, node.master)
	return nil
}

/**
* getDB
* @param name string
* @return *DB, error
**/
func (s *Methods) getDB(name string) (*DB, error) {
	var response DB
	err := callRpc(node.master, "Methods.GetDB", name, &response)
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
* SignIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func SignIn(device, database, username, password string) (*Session, error) {
	return signIn(device, database, username, password)
}

package rds

import (
	"fmt"

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

	var err error
	response, err = node.getDb(require)
	if err != nil {
		return err
	}

	return nil
}

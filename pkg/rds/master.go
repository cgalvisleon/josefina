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

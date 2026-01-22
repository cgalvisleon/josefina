package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Follow struct{}

var follow *Follow

/**
* ping
* @return error
**/
func (s *Follow) ping() error {
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
* Select
* @params require et.Json, response *et.Item
* @return error
**/
func (s *Follow) Select(require et.Json, response *et.Item) error {
	return nil
}

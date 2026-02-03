package jdb

import (
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/internal/jql"
	"github.com/cgalvisleon/josefina/internal/mod"
)

var (
	appName  string = "josefina"
	version  string = "0.0.1"
	node     *Node
	hostname string = ""
)

func init() {
	hostname, _ = os.Hostname()
}

/**
* Load: Initializes josefine
* @return error
**/
func Load() error {
	if node != nil {
		return nil
	}

	err := mod.Load(node.isStrict)
	if err != nil {
		return err
	}

	err = cache.Load(node.getLeader)
	if err != nil {
		return err
	}

	err = jql.Load(node.getLeader, node.nextHost, node.isStrict)
	if err != nil {
		return err
	}

	port := envar.GetInt("RPC_PORT", 4200)
	isStrict := envar.GetBool("IS_STRICT", false)
	node = newNode(hostname, port, isStrict)

	go node.start()

	return nil
}

/**
* HelpCheck: Returns the help check
* @return et.Item
**/
func HelpCheck() et.Item {
	if !node.started {
		return et.Item{
			Ok: false,
			Result: et.Json{
				"status":  false,
				"message": "josefina is not started",
			},
		}
	}

	return et.Item{
		Ok:     true,
		Result: node.helpCheck(),
	}
}

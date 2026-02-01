package jdb

import (
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

var (
	node        *Node
	packageName string
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	port := envar.GetInt("RPC_PORT", 4200)
	node = newNode(hostname, port)
}

/**
* Load: Initializes josefine
* @return error
**/
func Load() error {
	if node.started {
		return nil
	}

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

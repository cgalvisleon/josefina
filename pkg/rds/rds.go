package rds

import (
	"os"

	"github.com/cgalvisleon/et/envar"
)

var (
	packageName = "josefina"
	Version     = "0.0.1"
	node        *Node
	hostname    string
)

func init() {
	hostname, _ = os.Hostname()
	node = &Node{}
}

/**
* Load: Initializes josefine
* @param version string
* @return error
**/
func Load(version string) error {
	if node.started {
		return nil
	}

	port := envar.GetInt("RPC_PORT", 4200)
	node = newNode(hostname, port, version)
	go node.start()

	return nil
}

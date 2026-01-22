package rds

import (
	"os"

	"github.com/cgalvisleon/et/envar"
)

var (
	packageName = "josefina"
	node        *Node
	hostname    string
)

func init() {
	hostname, _ = os.Hostname()
}

/**
* Load: Initializes josefine
* @param version string
* @return error
**/
func Load(version string) error {
	if node != nil {
		return nil
	}

	leader, err := getLeader()
	if err != nil {
		return err
	}

	port := envar.GetInt("RPC_PORT", 4200)
	node = newNode(hostname, port, version)
	if node.host != leader {
		node.leader = leader
	}

	if methods == nil {
		methods = new(Methods)
	}
	err = node.mount(methods)
	if err != nil {
		return err
	}

	go node.start()

	if node.leader != "" {
		err = methods.ping(node.leader)
		if err != nil {
			return err
		}
	}

	return nil
}

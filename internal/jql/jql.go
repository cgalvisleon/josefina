package jql

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

var (
	node *Node
)

func init() {
	node = &Node{
		address: "",
		dbs:     make(map[string]*catalog.DB, 0),
	}
}

/**
* Load: Loads the cache
* @param getLeader func() (string, bool), isStrict bool
* @return error
**/
func Load(getLeader func() (string, bool), isStrict bool) error {
	node.getLeader = getLeader
	node.isStrict = isStrict

	hostname, _ := os.Hostname()
	port := envar.GetInt("RPC_PORT", 4200)
	address := fmt.Sprintf("%s:%d", hostname, port)

	_, err := jrpc.Mount(address, syn)
	if err != nil {
		logs.Panic(err)
	}

	node.address = address
	return nil
}

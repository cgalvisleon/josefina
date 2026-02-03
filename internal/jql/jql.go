package jql

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/dbs"
)

var (
	node *Node
)

func init() {
	node = &Node{
		address: "",
		dbs:     make(map[string]*dbs.DB, 0),
	}
}

/**
* Load: Loads the cache
* @param getLeader func() (string, bool)
* @return error
**/
func Load(getLeader func() (string, bool), getNextHost func() string, isStrict bool) error {
	node.getLeader = getLeader
	node.nextHost = getNextHost
	node.isStrict = isStrict

	hostname, _ := os.Hostname()
	port := envar.GetInt("RPC_PORT", 4200)
	address := fmt.Sprintf("%s:%d", hostname, port)

	syn = &Jql{}
	_, err := jrpc.Mount(address, syn)
	if err != nil {
		logs.Panic(err)
	}

	node.address = address
	return nil
}

func toQuery(query et.Json) (*Jql, error) {
	return &Jql{}, nil
}

func (s *Jql) run() (et.Items, error) {
	return et.Items{}, nil
}

package core

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
)

/**
* Load: Loads the cache
* @param fn func() (string, bool)
* @return error
**/
func Load(getLeader func() (string, bool)) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	port := envar.GetInt("RPC_PORT", 4200)
	address := fmt.Sprintf("%s:%d", hostname, port)
	_, err = jrpc.Mount(address, syn)
	if err != nil {
		logs.Panic(err)
	}

	syn.getLeader = getLeader
	syn.address = address
	return nil
}

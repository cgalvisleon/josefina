package cache

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
)

type MemCache struct{}

var (
	syn      *MemCache
	hostname string
)

func init() {
	syn = &MemCache{}
	_, err := jrpc.Mount(hostname, syn)
	if err != nil {
		logs.Panic(err)
	}

	hostname, _ = os.Hostname()
	port := envar.GetInt("RPC_PORT", 4200)
	hostname = fmt.Sprintf("%s:%d", hostname, port)
}

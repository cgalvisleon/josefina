package main

import (
	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/josefina/internal/services"
)

func main() {
	envar.SetIntByArg("-port", "PORT", 3300)
	envar.SetIntByArg("-rpc", "RPC_PORT", 4200)
	envar.SetIntByArg("-tcp", "TCP_PORT", 5200)
	envar.SetBoolByArg("-strict", "IS_STRICT", false)

	srv := serv.New()
	srv.Start()
}

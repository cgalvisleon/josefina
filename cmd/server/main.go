package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	serv "github.com/cgalvisleon/josefina/internal/services"
)

func main() {
	envar.SetIntByArg("-port", "PORT", 3300)
	envar.SetIntByArg("-rpc", "RPC_PORT", 4200)
	envar.SetIntByArg("-tcp", "TCP_PORT", 5200)
	envar.SetBoolByArg("-strict", "IS_STRICT", false)

	srv, err := serv.New()
	if err != nil {
		logs.Fatal(err)
	}

	srv.Start()
}

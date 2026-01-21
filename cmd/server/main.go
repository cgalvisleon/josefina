package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	serv "github.com/cgalvisleon/josefina/internal/services"
)

func main() {
	envar.SetIntByArg("port", "PORT", 3300)
	envar.SetIntByArg("rpc", "RPC_PORT", 4200)
	envar.SetStrByArg("mode", "MODE", "master")

	srv, err := serv.New()
	if err != nil {
		logs.Fatal(err)
	}

	srv.Start()
}

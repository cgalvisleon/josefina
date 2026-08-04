package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	serv "github.com/josefina/internal/server"
)

func main() {
	envar.SetIntByArg("port", "PORT", 1377)
	envar.SetIntByArg("rpct", "RPC_PORT", 4377)

	srv, err := serv.New()
	if err != nil {
		logs.Fatal(err)
	}

	srv.Start()
}

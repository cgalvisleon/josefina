package main

import (
	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/josefina/internal/server"
)

func main() {
	envar.SetIntByArg("-tcp-port", "TCP_PORT", 1377)
	envar.SetIntByArg("-http-port", "HTTP_PORT", 3500)
	envar.SetBoolByArg("-strict", "IS_STRICT", false)

	srv := serv.New()
	srv.Start()
}

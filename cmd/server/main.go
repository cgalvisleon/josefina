package main

import (
	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/josefina/internal/server"
)

func main() {
	envar.SetIntByArg("-port", "PORT", 1377)
	envar.SetIntByArg("-http", "HTTP", 3500)
	envar.SetBoolByArg("-strict", "IS_STRICT", false)

	port := envar.GetInt("PORT", 1377)
	srv := serv.New(port)
	srv.Start()
}

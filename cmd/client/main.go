package main

import (
	"github.com/cgalvisleon/et/envar"
	cli "github.com/cgalvisleon/josefina/internal/client"
)

func main() {
	envar.SetStrByArg("-data", "DATA_PATH", "./data")
	envar.SetStrByArg("-user", "USERNAME", "admin")
	envar.SetStrByArg("-password", "PASSWORD", "")
	envar.SetStrByArg("-database", "DATABASE", "")

	srv, err := cli.New()
	if err != nil {
		panic(err)
	}

	srv.Start()
}

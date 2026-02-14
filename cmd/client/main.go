package main

import (
	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/josefina/internal/client"
)

func main() {
	envar.SetStrByArg("-host", "HOST", "localhost:1377")
	envar.SetStrByArg("-user", "USER", "admin")
	envar.SetStrByArg("-password", "PASSWORD", "admin")
	envar.SetStrByArg("-database", "DATABASE", "josefina")

	srv := serv.New()
	srv.Start()
}

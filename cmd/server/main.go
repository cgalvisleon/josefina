package main

import (
	"github.com/cgalvisleon/et/envar"
	serv "github.com/cgalvisleon/josefina/internal/services"
)

func main() {
	envar.SetIntByArg("-port", "PORT", 3030)
	envar.SetBoolByArg("-strict", "IS_STRICT", false)

	srv := serv.New()
	srv.Start()
}

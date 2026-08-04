package main

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	srv "github.com/josefina/internal/server"
)

func main() {
	envar.SetIntByArg("port", "PORT", 1370)
	envar.SetIntByArg("rpct", "RPC_PORT", 4370)
	envar.SetStrByArg("name", "DB_NAME", "josefina")
	envar.SetStrByArg("path_data", "DB_PATH_DATA", "./data")
	envar.SetStrByArg("path_wald", "DB_PATH_WALD", "./data")

	srv, err := srv.New()
	if err != nil {
		logs.Fatal(err)
	}

	srv.Start()
}

package main

import (
	"github.com/cgalvisleon/et/logs"
	serv "github.com/cgalvisleon/josefina/internal/services"
)

func main() {
	srv, err := serv.New()
	if err != nil {
		logs.Fatal(err)
	}

	srv.Start()
}

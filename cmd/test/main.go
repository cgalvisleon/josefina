package main

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/jdb"
)

const dataPath = "./data/test"

func main() {
	os.RemoveAll(dataPath)

	if err := run(); err != nil {
		logs.Fatal(err)
	}
}

func run() error {
	server, err := jdb.Load()
	if err != nil {
		return err
	}

	fmt.Println(server.Version)
	return nil
}

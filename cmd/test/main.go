package main

import (
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
	db, err := jdb.NewDb("./data", "test")
	if err != nil {
		return err
	}
	if err := db.Init(); err != nil {
		return err
	}

	return nil
}

package rds

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
)

var (
	packageName = "josefina"
	node        *Node
	hostname    string
)

func init() {
	hostname, _ = os.Hostname()
}

/**
* Load: Initializes josefine
* @param version string
* @return error
**/
func Load(version string) error {
	if node != nil {
		return nil
	}

	port := envar.GetInt("RPC_PORT", 4200)
	master := envar.GetStr("MASTER_HOST", "")
	node = newNode(hostname, port, version)
	node.master = master

	if methods == nil {
		methods = new(Methods)
	}
	err := node.mount(methods)
	if err != nil {
		return err
	}

	go node.start()

	if node.master != "" {
		err = methods.ping()
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* GetModel: Returns a model by database, schema and name
* @param database string, schema string, model string
* @return *Model, error
**/
func getModel(database, schema, model string) (*Model, error) {
	if node == nil {
		return nil, fmt.Errorf("node not initialized")
	}

	if node.Type == FOLLOW {
		return follow.getModel(database, schema, model)
	}

	db, err := getDB(database)
	if err != nil {
		return nil, err
	}

	return db.getModel(schema, model)
}

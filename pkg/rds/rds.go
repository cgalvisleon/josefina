package rds

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

var (
	packageName = "josefina"
	node        *Node
	hostName    string
)

func init() {
	hostName, _ = os.Hostname()
}

/**
* LoadMaster: Initializes the josefina
* @return error
**/
func LoadMaster(version string) error {
	if node != nil {
		return nil
	}

	path := envar.GetStr("TENNANT_PATH_DATA", "./data")
	node = newNode(Master, version, path)
	db := newDb(node.Path, packageName, node.Version)
	if err := initTransactions(db); err != nil {
		return err
	}
	if err := initDatabases(db); err != nil {
		return err
	}
	if err := initUsers(db); err != nil {
		return err
	}
	if err := initSeries(db); err != nil {
		return err
	}
	if err := initRecords(db); err != nil {
		return err
	}
	if err := initModels(db); err != nil {
		return err
	}

	return nil
}

/**
* Follow: Initializes the josefina as a follow node
* @param version string
* @return error
**/
func LoadFollow(version string) error {
	if node != nil {
		return nil
	}

	path := envar.GetStr("TENNANT_PATH_DATA", "./data")
	node = newNode(Master, version, path)
	return nil
}

/**
* getDB: Returns a database by name
* @param name string
* @return *DB, error
**/
func getDB(name string) (*DB, error) {
	if node == nil {
		return nil, fmt.Errorf("node not initialized")
	}

	result, err := node.getDb(name)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* GetModel: Returns a model by database, schema and name
* @param database string, schema string, model string
* @return *Model, error
**/
func getModel(database, schema, model string) (*Model, error) {
	db, err := getDB(database)
	if err != nil {
		return nil, err
	}

	return db.getModel(schema, model)
}

/**
* Select: Returns a records that complies with the query
* @param query et.Json
* @return et.Items, error
**/
func Select(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* Insert: Inserts a record
* @param model string, data []et.Json
* @return et.Items, error
**/
func Insert(model string, data []et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* Update: Updates a record
* @param model string, data et.Json, where et.Json
* @return et.Items, error
**/
func Update(model string, data et.Json, where et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* Delete: Deletes a record
* @param model string, where et.Json
* @return et.Items, error
**/
func Delete(model string, where et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* Upsert: Upserts a record
* @param model string, data et.Json, where et.Json
* @return et.Items, error
**/
func Upsert(model string, data et.Json, where et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* Define: Defines a model
* @param define et.Json
* @return et.Items, error
**/
func Define(define et.Json) (et.Items, error) {
	return et.Items{}, nil
}

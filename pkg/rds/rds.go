package rds

import (
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

var (
	tennant  *Tennant
	hostName string
)

func init() {
	hostName, _ = os.Hostname()
}

/**
* Init: Initializes the josefina
* @return error
**/

func Init(version string) error {
	path := envar.GetStr("TENNANT_PATH_DATA", "./data")
	name := envar.GetStr("TENNANT_NAME", "josefina")
	var err error
	tennant, err = loadTennant(path, name, version)
	if err != nil {
		return err
	}

	return nil
}

/**
* GetDB: Returns a database by name
* @param name string
* @return *DB, error
**/
func GetDB(name string) (*DB, error) {
	result, err := tennant.getDb(name)
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
func GetModel(database, schema, model string) (*Model, error) {
	db, err := GetDB(database)
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

package db

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/server/msg"
)

/**
* NewDatabase: Creates a new database
* @param name string, version int, release int
* @return *DB, error
**/
func NewDatabase(name string, version int, release int) (*DB, error) {
	if tennant == nil {
		return nil, fmt.Errorf(msg.MSG_TENNANT_NOT_FOUND)
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := tennant.Dbs[name]
	if ok {
		return result, nil
	}

	result = &DB{
		Name:    name,
		Version: version,
		Release: release,
		Path:    fmt.Sprintf("%s/%s", tennant.Path, name),
		Schemas: make(map[string]*Schema),
	}
	tennant.Dbs[name] = result

	return result, nil
}

/**
* GetDB: Returns a database by name
* @param name string
* @return *DB, error
**/
func GetDB(name string) (*DB, error) {
	result, err := NewDatabase(name, 1, 0)
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

/**
* Query: Returns a records that complies with the query
* @param query et.Json
* @return et.Items, error
**/
func Query(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

package db

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/server/msg"
)

/**
* GetModel: Returns a model by database, schema and name
* @param database string, schema string, model string
* @return *Model, error
**/
func GetModel(database, schema, model string) (*Model, error) {
	db, ok := dbs[database]
	if !ok {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
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

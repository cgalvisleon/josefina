package jdb

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
)

var (
	ErrNotInserted = fmt.Errorf("record not inserted")
	ErrNotUpdated  = fmt.Errorf("record not updated")
	ErrNotFound    = fmt.Errorf("record not found")
	ErrNotUpserted = fmt.Errorf("record not inserted or updated")
	ErrDuplicate   = fmt.Errorf("record duplicate")
)

/**
* ConnectTo
* @param id, name, driver string, userCore bool, params Connection
* @return (*DB, error)
**/
func ConnectTo(id, name, driver string, userCore bool, params et.Json) (*DB, error) {
	return getDatabase(id, name, driver, userCore, params)
}

/**
* LoadTo
* @param name string
* @return (*DB, error)
**/
func LoadTo(name string) (*DB, error) {
	driver := envar.GetStr("DB_DRIVER", "postgres")
	params := et.Json{
		"database": name,
		"host":     envar.GetStr("DB_HOST", "localhost"),
		"port":     envar.GetInt("DB_PORT", 5432),
		"username": envar.GetStr("DB_USERNAME", "test"),
		"password": envar.GetStr("DB_PASSWORD", "test"),
		"app":      envar.GetStr("DB_APP", "test"),
		"version":  envar.GetInt("DB_VERSION", 15),
	}

	return getDatabase(name, name, driver, true, params)
}

/**
* Load
* @return (*DB, error)
**/
func Load() (*DB, error) {
	name := envar.GetStr("DB_NAME", "josephine")
	return LoadTo(name)
}

/**
* GetDatabase
* @param name string
* @return (*DB, error)
**/
func GetDatabase(name string) (*DB, error) {
	idx := indexDatabase(name)
	if idx == -1 {
		return nil, fmt.Errorf(MSG_DATABASE_NOT_FOUND, name)
	}

	return databases[idx], nil
}

/**
* GetModel
* @param database, name string
* @return (*Model, error)
**/
func GetModel(database, name string) (*Model, error) {
	db, err := GetDatabase(database)
	if err != nil {
		return nil, err
	}

	result, err := db.GetModel(name)
	if err != nil {
		return nil, fmt.Errorf(MSG_MODEL_NOT_FOUND, name)
	}

	return result, nil
}

/**
* Define
* @param definition et.Json
* @return (*Model, error)
**/
func Define(definition et.Json) (*Model, error) {
	database := definition.String("database")
	if !utility.ValidStr(database, 0, []string{}) {
		return nil, fmt.Errorf(MSG_ATTRIBUTE_REQUIRED, "database")
	}

	db, err := GetDatabase(database)
	if err != nil {
		return nil, err
	}

	return db.Define(definition)
}

/**
* Query
* @param query et.Json
* @return (et.Items, error)
**/
func Query(query et.Json) (et.Items, error) {
	database := query.String("database")
	if !utility.ValidStr(database, 0, []string{}) {
		return et.Items{}, fmt.Errorf(MSG_ATTRIBUTE_REQUIRED, "database")
	}

	db, err := GetDatabase(database)
	if err != nil {
		return et.Items{}, err
	}

	insert := query.Json("insert")
	if !insert.IsEmpty() {
		return db.Insert(insert)
	}

	update := query.Json("update")
	if !update.IsEmpty() {
		return db.Update(update)
	}

	delete := query.Json("delete")
	if !delete.IsEmpty() {
		return db.Delete(delete)
	}

	upsert := query.Json("upsert")
	if !upsert.IsEmpty() {
		return db.Upsert(upsert)
	}

	return db.Select(query)
}

/**
* From
* @param model *Model, as string
* @return *Ql
**/
func From(model *Model, as string) *Ql {
	return newQuery(model, as, TpSelect)
}

/**
* HttpDefine
* @param w http.ResponseWriter, r *http.Request
* @return
**/
func HttpDefine(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := Define(body)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result.ToJson())
}

/**
* HttpQuery
* @param w http.ResponseWriter, r *http.Request
* @return
**/
func HttpQuery(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := Query(body)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

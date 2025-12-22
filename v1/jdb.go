package jdb

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/jdb"
)

const (
	DriverPostgres = jdb.DriverPostgres
	DriverSqlite   = jdb.DriverSqlite
	// Field names
	SOURCE     = jdb.SOURCE
	KEY        = jdb.KEY
	STATUS     = jdb.STATUS
	VERSION    = jdb.VERSION
	PROJECT_ID = jdb.PROJECT_ID
	TENANT_ID  = jdb.TENANT_ID
	CREATED_AT = jdb.CREATED_AT
	UPDATED_AT = jdb.UPDATED_AT
	// Data types
	TpAny      = jdb.TpAny
	TpBytes    = jdb.TpBytes
	TpInt      = jdb.TpInt
	TpFloat    = jdb.TpFloat
	TpKey      = jdb.TpKey
	TpText     = jdb.TpText
	TpJson     = jdb.TpJson
	TpDateTime = jdb.TpDateTime
	TpBoolean  = jdb.TpBoolean
	TpGeometry = jdb.TpGeometry
	TpCalc     = jdb.TpCalc
	// Column types
	TpColumn      = jdb.TpColumn
	TpAtrib       = jdb.TpAtrib
	TpDetail      = jdb.TpDetail
	TpRollup      = jdb.TpRollup
	TpAggregation = jdb.TpAggregation
	// Status record
	Active    = jdb.Active
	Archived  = jdb.Archived
	Canceled  = jdb.Canceled
	OfSystem  = jdb.OfSystem
	ForDelete = jdb.ForDelete
	Pending   = jdb.Pending
	Approved  = jdb.Approved
	Rejected  = jdb.Rejected
)

var ()

type TypeColumn = jdb.TypeColumn
type TypeData = jdb.TypeData
type Driver = jdb.Driver
type DB = jdb.DB
type Model = jdb.Model
type Tx = jdb.Tx
type Condition = jdb.Condition
type Ql = jdb.Ql
type Cmd = jdb.Cmd

/**
* NewModel
* @param db *DB, schema, name string, version int
* @return (*Model, error)
**/
func NewModel(db *DB, schema, name string, version int) (*Model, error) {
	result, err := db.Define(et.Json{
		"schema":  schema,
		"name":    name,
		"version": version,
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* ConnectTo
* @param tenantId, name, driver string, userCore bool, params et.Json
* @return (*DB, error)
**/
func ConnectTo(tenantId, name, driver string, userCore bool, params et.Json) (*DB, error) {
	return jdb.ConnectTo(tenantId, name, driver, userCore, params)
}

/**
* LoadTo
* @param name string
* @return (*DB, error)
**/
func LoadTo(name string) (*DB, error) {
	return jdb.LoadTo(name)
}

/**
* Load
* @return (*DB, error)
**/
func Load() (*DB, error) {
	return jdb.Load()
}

/**
* Define
* @param definition et.Json
* @return (*Model, error)
**/
func Define(definition et.Json) (*Model, error) {
	return jdb.Define(definition)
}

/**
* From
* @param model *Model, as string
* @return *Ql
**/
func From(model *Model, as string) *Ql {
	return jdb.From(model, as)
}

/**
* Query
* @param query et.Json
* @return et.Items, error
**/
func Query(query et.Json) (et.Items, error) {
	return jdb.Query(query)
}

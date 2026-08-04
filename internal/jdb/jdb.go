package jdb

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

var databases map[string]*DB

func init() {
	databases = make(map[string]*DB, 0)
}

const (
	sysSchema  = "catalog"
	sysCatalog = sysSchema
)

/**
* NewDb: Creates a new database
* @param pathData, pathWald, name string
* @return *DB, error
**/
func NewDb(pathData, pathWald, name string) (*DB, error) {
	name = store.Normalize(name)
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	if pathData == "" {
		pathData = "./data"
	}
	if pathWald == "" {
		pathWald = pathData
	}
	result := &DB{
		Name:                name,
		PathData:            pathData,
		PathWald:            pathWald,
		Lang:                "en",
		TransactionTTL:      10 * time.Second,
		RelSegSize:          1024,
		SyncOnWrite:         false,
		TennantName:         "",
		TennantPathData:     "",
		Timezone:            "America/Bogota",
		MinThresholdCompact: 100,
		schemas:             make(map[string]*Schema, 0),
		mu:                  &sync.RWMutex{},
	}

	var err error
	result.store, err = result.loadModel(sysSchema, sysCatalog, 1, true)
	if err != nil {
		return nil, err
	}

	result.cache, err = result.loadCache()
	if err != nil {
		return nil, err
	}

	result.users, err = result.loadUsers()
	if err != nil {
		return nil, err
	}

	result.sessions, err = result.loadSessions()
	if err != nil {
		return nil, err
	}

	databases[name] = result
	return result, nil
}

/**
* loadDb: Loads a database from the JSON definition
* @param pathData, pathWald, name string
* @return *DB, error
**/
func loadDb(pathData, pathWald, name string) (*DB, error) {
	name = store.Normalize(name)
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	result, exists := databases[name]
	if exists {
		return result, nil
	}

	result, err := NewDb(pathData, pathWald, name)
	if err != nil {
		return nil, err
	}

	if err := result.Init(); err != nil {
		return nil, err
	}

	err = result.Load()
	if err != nil {
		return nil, err
	}

	databases[name] = result
	return result, nil
}

/**
* Load: Loads the database from the JSON definition
* @return error
**/
func Load() (*DB, error) {
	name := envar.GetStr("DB_NAME", "josefina")
	pathData := envar.GetStr("DB_PATH_DATA", "./data")
	pathWald := envar.GetStr("DB_PATH_WALD", "./data")
	return loadDb(pathData, pathWald, name)
}

/**
* GetDb: Returns the database by name
* @param name string
* @return *DB, error
**/
func GetDb(name string) (*DB, error) {
	db, exists := databases[name]
	if !exists {
		return nil, fmt.Errorf(msg.MSG_DB_NOT_FOUND, name)
	}

	return db, nil
}

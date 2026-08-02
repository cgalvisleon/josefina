package jdb

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

var databases map[string]*DB

func init() {
	databases = make(map[string]*DB, 0)
}

/**
* NewDb: Creates a new database
* @param path, name string
* @return *DB, error
**/
func NewDb(path, name string) (*DB, error) {
	name = store.Normalize(name)
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	path = filepath.Join(path, name)
	result := &DB{
		Name: name,
		Path: path,
		Config: &Config{
			IsStrict:            false,
			Lang:                "en",
			TransactionTTL:      10 * time.Second,
			RelSegSize:          1024,
			SyncOnWrite:         false,
			TennantName:         "",
			TennantPathData:     "",
			Timezone:            "America/Bogota",
			MinThresholdCompact: 100,
		},
		schemas: make(map[string]*Schema, 0),
		mu:      &sync.RWMutex{},
	}

	databases[name] = result

	return result, nil
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

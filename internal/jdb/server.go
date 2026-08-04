package jdb

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

const (
	defaultDbName = "josefina"
	sysSchema     = "catalog"
	sysCatalog    = sysSchema
)

type Server struct {
	Version       string           `json:"version"`
	PathDatabases string           `json:"path_databases"`
	PathWal       string           `json:"path_wal"`
	PathSystem    string           `json:"path_system"`
	dbs           map[string]*DB   `json:"-"`
	mu            *sync.RWMutex    `json:"-"`
	store         *store.FileStore `json:"-"`
	sessions      *Sessions        `json:"-"` // Sessions
}

var server *Server

/**
* load: Loads the server
* @return error
**/
func (s *Server) load() error {
	s.store.ForEach(func(id string, data []byte) (bool, error) {
		var item et.Json
		if err := json.Unmarshal(data, &item); err != nil {
			return false, err
		}

		err := s.loadDb(item)
		if err != nil {
			return false, err
		}

		return true, nil
	}, true, 0, 0)

	if len(s.dbs) == 0 {
		db, err := s.newDb(defaultDbName)
		if err != nil {
			return err
		}

		err = db.save()
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

/**
* AddDb: Adds a database to the server
* @param db *DB
**/
func (s *Server) addDb(db *DB) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dbs[db.Name] = db
}

/**
* getDb: Gets a database from the server
* @param name string
* @return *DB, error
**/
func (s *Server) getDb(name string) (*DB, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db, exists := s.dbs[name]
	return db, exists
}

/**
* removeDb: Removes a database from the server
* @param name string
**/
func (s *Server) removeDb(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dbs, name)
}

/**
* newDb: Creates a new database
* @param name string
* @return *DB, error
**/
func (s *Server) newDb(name string) (*DB, error) {
	name = store.Normalize(name)
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	pathDatabases := filepath.Join(s.PathDatabases, name)
	pathWal := filepath.Join(s.PathWal, name)
	result := &DB{
		Name:                name,
		PathDatabases:       pathDatabases,
		PathWal:             pathWal,
		Lang:                "en",
		TransactionTTL:      10 * time.Second,
		RelSegSize:          1024,
		SyncOnWrite:         false,
		TennantName:         "",
		TennantPathData:     "",
		Timezone:            "America/Bogota",
		MinThresholdCompact: 100,
		Version:             s.Version,
		server:              s,
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

	s.addDb(result)

	err = result.init()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* loadDb: Loads a database from the JSON definition
* @param params et.Json
* @return error
**/
func (s *Server) loadDb(params et.Json) error {
	name := params.Str("name")
	result, exists := s.getDb(name)
	if exists {
		return nil
	}

	result = &DB{
		Name:                name,
		PathDatabases:       params.Str("path_databases"),
		PathWal:             params.Str("path_wal"),
		Lang:                params.Str("lang"),
		TransactionTTL:      params.ValDuration(10*time.Second, "transaction_ttl"),
		RelSegSize:          params.Int("rel_seg_size"),
		SyncOnWrite:         params.Bool("sync_on_write"),
		TennantName:         params.Str("tennant_name"),
		TennantPathData:     params.Str("tennant_path_data"),
		Timezone:            params.Str("timezone"),
		MinThresholdCompact: params.Int("min_threshold_compact"),
		Version:             params.Str("version"),
		server:              s,
		schemas:             make(map[string]*Schema, 0),
		mu:                  &sync.RWMutex{},
	}

	var err error
	result.store, err = result.loadModel(sysSchema, sysCatalog, 1, true)
	if err != nil {
		return err
	}

	result.cache, err = result.loadCache()
	if err != nil {
		return err
	}

	result.users, err = result.loadUsers()
	if err != nil {
		return err
	}

	schemas := params.ArrayJson("schemas")
	for _, schemaDef := range schemas {
		name := schemaDef.Str("name")
		sch, exists := result.getSchema(name)
		if !exists {
			var err error
			sch, err = result.newSchema(name)
			if err != nil {
				return err
			}
		}

		models := schemaDef.Json("models")
		for name := range models {
			modelDef := models.Json(name)
			_, exists := sch.getModel(name)
			if !exists {
				_, err := sch.loadModel(modelDef)
				if err != nil {
					return err
				}
			}
		}
	}

	s.addDb(result)

	err = result.init()
	if err != nil {
		return err
	}

	return nil
}

/**
* saveDb: Saves a database to the JSON definition
* @param db *DB
* @return error
**/
func (s *Server) saveDb(db *DB) error {
	_, _, err := s.store.Put(db.Name, db.ToJson())
	if err != nil {
		return err
	}

	return nil
}

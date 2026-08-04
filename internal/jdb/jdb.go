package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
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
	Version    string           `json:"version"`
	PathData   string           `json:"path_data"`
	PathWald   string           `json:"path_wald"`
	PathSystem string           `json:"path_system"`
	dbs        map[string]*DB   `json:"-"`
	mu         *sync.RWMutex    `json:"-"`
	store      *store.FileStore `json:"-"`
}

var server *Server

/**
* Load: Loads the server
* @return *Server
**/
func Load() (*Server, error) {
	if server != nil {
		return server, nil
	}

	server = &Server{
		Version:    "1.0.0",
		PathData:   envar.GetStr("DB_PATH_DATA", "./data/collection"),
		PathWald:   envar.GetStr("DB_PATH_WALD", "./data/wald"),
		PathSystem: envar.GetStr("DB_PATH_SYSTEM", "./data/system"),
		dbs:        make(map[string]*DB),
		mu:         &sync.RWMutex{},
	}

	var err error
	server.store, err = store.Open(server.PathSystem, server.PathSystem, "system", store.ReadWrite)
	if err != nil {
		return nil, err
	}

	err = server.load()
	if err != nil {
		return nil, err
	}

	return server, nil
}

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

	pathData := filepath.Join(s.PathData, name)
	pathWald := filepath.Join(s.PathWald, name)
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

	result.sessions, err = result.loadSessions()
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
	// name = store.Normalize(name)
	// if !utility.ValidStr(name, 0, []string{""}) {
	// 	return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	// }

	// result, exists := s.getDb(name)
	// if exists {
	// 	return result, nil
	// }

	// s.Lang = def.Str("lang")
	// s.TransactionTTL = def.ValDuration(10*time.Second, "transaction_ttl")
	// s.RelSegSize = def.Int("rel_seg_size")
	// s.SyncOnWrite = def.Bool("sync_on_write")
	// s.TennantName = def.Str("tennant_name")
	// s.TennantPathData = def.Str("tennant_path_data")
	// s.Timezone = def.Str("timezone")
	// s.MinThresholdCompact = def.Int("min_threshold_compact")

	// result, err := s.NewDb(pathData, pathWald, name)
	// if err != nil {
	// 	return nil, err
	// }

	// if err := result.init(); err != nil {
	// 	return nil, err
	// }

	// err = result.load()
	// if err != nil {
	// 	return nil, err
	// }

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

/**
* NewDb: Creates a new database
* @param name string
* @return *DB, error
**/
func NewDb(name string) (*DB, error) {
	if server == nil {
		return nil, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	return server.newDb(name)
}

/**
* GetDb: Returns the database by name
* @param name string
* @return *DB, error
**/
func GetDb(name string) (*DB, error) {
	if server == nil {
		return nil, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	db, exists := server.getDb(name)
	if !exists {
		return nil, fmt.Errorf(msg.MSG_DB_NOT_FOUND, name)
	}

	return db, nil
}

func DeleteDb(name string) error {
	if server == nil {
		return errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	db, exists := server.getDb(name)
	if !exists {
		return fmt.Errorf(msg.MSG_DB_NOT_FOUND, name)
	}

	err := db.Empty()
	if err != nil {
		return err
	}

	_, err = server.store.Delete(name)
	if err != nil {
		return err
	}

	server.removeDb(name)

	return nil
}

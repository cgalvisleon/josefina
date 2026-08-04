package jdb

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

const (
	sysSchema  = "catalog"
	sysCatalog = sysSchema
)

type Server struct {
	Version string           `json:"version"`
	dbs     map[string]*DB   `json:"-"`
	mu      *sync.RWMutex    `json:"-"`
	store   *store.FileStore `json:"-"`
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
		Version: "1.0.0",
		dbs:     make(map[string]*DB),
		mu:      &sync.RWMutex{},
	}

	var err error
	path := envar.GetStr("DB_PATH_SYSTEM", "./data/system")
	server.store, err = store.Open(path, path, "system", store.ReadWrite)
	if err != nil {
		return nil, err
	}

	return server, nil
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
* NewDb: Creates a new database
* @param pathData, pathWald, name string
* @return *DB, error
**/
func (s *Server) NewDb(pathData, pathWald, name string) (*DB, error) {
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
	pathData = filepath.Join(pathData, name)
	pathWald = filepath.Join(pathWald, name)
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
	return result, nil
}

/**
* loadDb: Loads a database from the JSON definition
* @param pathData, pathWald, name string
* @return *DB, error
**/
func (s *Server) loadDb(pathData, pathWald, name string) (*DB, error) {
	name = store.Normalize(name)
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	result, exists := s.getDb(name)
	if exists {
		return result, nil
	}

	result, err := s.NewDb(pathData, pathWald, name)
	if err != nil {
		return nil, err
	}

	if err := result.Init(); err != nil {
		return nil, err
	}

	err = result.load()
	if err != nil {
		return nil, err
	}

	return result, nil
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

package jdb

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

/**
* Load: Loads the server
* @return *Server
**/
func Load() (*Server, error) {
	if server != nil {
		return server, nil
	}

	server = &Server{
		Version:       "1.0.0",
		PathDatabases: envar.GetStr("DB_PATH_DATA", "./data/databases"),
		PathWal:       envar.GetStr("DB_PATH_WAL", "./data/wal"),
		PathSystem:    envar.GetStr("DB_PATH_SYSTEM", "./data/system"),
		dbs:           make(map[string]*DB),
		mu:            &sync.RWMutex{},
	}

	var err error
	server.store, err = store.Open(server.PathSystem, server.PathSystem, "system", store.ReadWrite)
	if err != nil {
		return nil, err
	}

 err = server.loadSessions()
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
* CreateDb: Creates a new database
* @param name string
* @return *DB, error
**/
func CreateDb(name string) (*DB, error) {
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

/**
* DeleteDb: Deletes a database
* @param name string
* @return error
**/
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

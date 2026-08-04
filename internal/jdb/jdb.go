package jdb

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
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
	if exists {
		return db, nil
	}

	result, err := server.loadDb(name)
	if err != nil {
		return nil, err
	}

	return result, nil
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

/**
* SignIn: Signs in a user
* @param database, username, password string
* @return et.Item, error
**/
func SignIn(database, username, password string) (et.Item, error) {
	db, err := GetDb(database)
	if err != nil {
		return et.Item{}, err
	}

	user, err := db.getUser(username)
	if err != nil {
		return et.Item{}, err
	}

	hash, err := utility.HashSHA512(password)
	if err != nil {
		return et.Item{}, err
	}

	if user.Password != hash {
		return et.Item{}, errors.New(msg.MSG_INVALID_PASSWORD)
	}

	device := "apiRest"
	duration := time.Minute * 60
	token, err := claim.NewToken(appName, device, user.ID, user.Username, user.Password, db.Name, et.Json{}, duration)
	if err != nil {
		return et.Item{}, err
	}

	_, err = server.newSession(token, db, HTTP, et.Json{})
	if err != nil {
		return et.Item{}, err
	}

	return et.Item{
		Ok: true,
		Result: et.Json{
			"token": token,
		},
	}, nil
}

/**
* GetSession: Gets a session from the server
* @param token string
* @return *Session, error
**/
func getSession(token string) (*Session, error) {
	if server == nil {
		return nil, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.getSession(token)
	if err != nil {
		return nil, err
	}

	if result.IsExpired() {
		err = server.removeSession(token)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(msg.MSG_SESSION_EXPIRED)
	}

	return result, nil
}

/**
* JQuery: Executes a query
* @param database string, query et.Json
* @return et.Items, error
**/
func JQuery(database string, query et.Json) (et.Items, error) {
	db, err := GetDb(database)
	if err != nil {
		return et.Items{}, err
	}

	result, err := db.JQuery(query)
	if err != nil {
		return et.Items{}, err
	}
	return result, nil
}

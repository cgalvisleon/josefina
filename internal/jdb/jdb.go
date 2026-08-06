package jdb

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
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

	err := event.Load()
	if err != nil {
		return nil, err
	}

	err = cache.Load()
	if err != nil {
		return nil, err
	}

	pool := envar.GetInt("DB_POOL", 0)
	if pool == 0 {
		pool = runtime.NumCPU()*2 + 1
	}
	server = &Server{
		Version:       "1.0.0",
		PathDatabases: envar.GetStr("DB_PATH_DATA", "./data/databases"),
		PathWal:       envar.GetStr("DB_PATH_WAL", "./data/wal"),
		PathSystem:    envar.GetStr("DB_PATH_SYSTEM", "./data/system"),
		dbs:           make(map[string]*DB),
		mu:            &sync.RWMutex{},
		request:       make(chan *Request, pool*4),
		pool:          pool,
	}
	server.runWorkers()

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
* Authenticate: Authenticates a session from the server
* @param token string
* @return *Session, error
**/
func Authenticate(token string) (*Session, error) {
	if server == nil {
		return nil, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.getSession(token)
	if err != nil {
		return nil, err
	}

	if result.IsExpired() {
		err = server.deleteSession(token)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(msg.MSG_SESSION_EXPIRED)
	}

	return result, nil
}

/**
* SignIn: Signs in a user
* @param database, username, password string
* @return et.Item, error
**/
func SignIn(database, username, password string) (et.Item, error) {
	if server == nil {
		return et.Item{}, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.Exec(et.Json{
		"database": database,
		"username": username,
		"password": password,
	}, server.signin)
	if err != nil {
		return et.Item{}, err
	}

	return result.First()
}

/**
* SignOut: Signs out a session
* @param token string
* @return error
**/
func SignOut(token string) (et.Items, error) {
	if server == nil {
		return et.Items{}, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.Exec(et.Json{
		"token": token,
	}, func(query et.Json) (et.Items, error) {
		token := query.Str("token")
		err := server.deleteSession(token)
		if err != nil {
			return et.Items{}, err
		}
		return et.Items{
			Ok:    true,
			Count: 1,
			Result: []et.Json{
				{
					"message": "Signed out successfully",
				},
			},
		}, nil
	})

	return result, err
}

/**
* System: Executes a system command
* @param query []et.Json
* @return et.Items, error
**/
func System(params et.Json) (et.Items, error) {
	if server == nil {
		return et.Items{}, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.Exec(params, server.jSystem)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* JQuery: Executes a query
* @param database string, query et.Json
* @return et.Items, error
**/
func JQuery(query et.Json) (et.Items, error) {
	if server == nil {
		return et.Items{}, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.Exec(query, server.jQuery)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* JCommand: Executes a command
* @param command et.Json
* @return et.Items, error
**/
func JCommand(command et.Json) (et.Items, error) {
	if server == nil {
		return et.Items{}, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.Exec(command, server.jCommand)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* JUploadXls: Uploads an XLS file
* @param params et.Json
* @return et.Items, error
**/
func JUploadXls(params et.Json) (et.Items, error) {
	if server == nil {
		return et.Items{}, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.Exec(params, server.jUploadXls)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* JUploadCsv: Uploads a CSV file
* @param params et.Json
* @return et.Items, error
**/
func JUploadCsv(params et.Json) (et.Items, error) {
	if server == nil {
		return et.Items{}, errors.New(msg.MSG_SERVER_NOT_LOADED)
	}

	result, err := server.Exec(params, server.jUploadCsv)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

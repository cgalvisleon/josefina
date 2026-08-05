package jdb

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

const (
	appName       = "josefina"
	defaultDbName = appName
	sysSchema     = "catalog"
	sysCatalog    = sysSchema
)

/**
* Request: A queued execution awaiting a worker from the pool.
**/
type Request struct {
	params et.Json
	fn     func(params et.Json) (et.Items, error)
	result chan Response
}

/**
* Response: The outcome of a queued Request.
**/
type Response struct {
	Items et.Items
	Err   error
}

type Server struct {
	Version       string           `json:"version"`
	PathDatabases string           `json:"path_databases"`
	PathWal       string           `json:"path_wal"`
	PathSystem    string           `json:"path_system"`
	dbs           map[string]*DB   `json:"-"`
	mu            *sync.RWMutex    `json:"-"`
	store         *store.FileStore `json:"-"`
	sessions      *Sessions        `json:"-"` // Sessions
	request       chan *Request    `json:"-"` // Queue where all executions arrive
	pool          int              `json:"-"` // Worker pool size: cores*2 + 1
}

var server *Server

/**
* runWorkers: Starts the pool of workers consuming the request queue.
**/
func (s *Server) runWorkers() {
	for i := 0; i < s.pool; i++ {
		go func() {
			for req := range s.request {
				items, err := req.fn(req.params)
				req.result <- Response{Items: items, Err: err}
			}
		}()
	}
}

/**
* Exec: Queues fn with query on the request queue and blocks until a worker executes it.
* @param query et.Json, fn func(query et.Json) (et.Items, error)
* @return et.Items, error
**/
func (s *Server) Exec(params et.Json, fn func(params et.Json) (et.Items, error)) (et.Items, error) {
	req := &Request{params: params, fn: fn, result: make(chan Response, 1)}
	s.request <- req
	res := <-req.result
	return res.Items, res.Err
}

/**
* load: Loads the server
* @return error
**/
func (s *Server) load() error {
	if s.store.Count() == 0 {
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
* @return *DB, bool
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
		TransactionTTL:      60 * time.Minute,
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

	err = result.loadCache()
	if err != nil {
		return nil, err
	}

	err = result.loadUsers()
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
func (s *Server) loadDb(name string) (*DB, error) {
	name = store.Normalize(name)
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	exists, params, err := s.store.GetObject(name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
	}

	result := &DB{
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

	result.store, err = result.loadModel(sysSchema, sysCatalog, 1, true)
	if err != nil {
		return nil, err
	}

	err = result.loadCache()
	if err != nil {
		return nil, err
	}

	err = result.loadUsers()
	if err != nil {
		return nil, err
	}

	schemas := params.ArrayJson("schemas")
	for _, schemaDef := range schemas {
		name := schemaDef.Str("name")
		sch, exists := result.getSchema(name)
		if !exists {
			var err error
			sch, err = result.newSchema(name)
			if err != nil {
				return nil, err
			}
		}

		models := schemaDef.Json("models")
		for name := range models {
			modelDef := models.Json(name)
			_, exists := sch.getModel(name)
			if !exists {
				_, err := sch.loadModel(modelDef)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	err = result.init()
	if err != nil {
		return nil, err
	}

	s.addDb(result)

	return result, nil
}

/**
* saveDb: Saves a database to the JSON definition
* @param db *DB
* @return error
**/
func (s *Server) saveDb(db *DB) error {
	_, _, err := s.store.PutObject(db.Name, db.ToJson())
	if err != nil {
		return err
	}

	return nil
}

/**
* signin: Signs in a user
* @param database, username, password string
* @return et.Items, error
**/
func (s *Server) signin(params et.Json) (et.Items, error) {
	database := params.Str("database")
	username := params.Str("username")
	password := params.Str("password")

	if database == "" {
		return et.Items{}, errors.New(msg.MSG_DATABASE_IS_REQUIRED)
	}

	if username == "" {
		return et.Items{}, errors.New(msg.MSG_USERNAME_IS_REQUIRED)
	}

	if password == "" {
		return et.Items{}, errors.New(msg.MSG_PASSWORD_IS_REQUIRED)
	}

	db, err := s.loadDb(database)
	if err != nil {
		return et.Items{}, err
	}

	user, err := db.getUser(username)
	if err != nil {
		return et.Items{}, err
	}

	hash, err := utility.HashSHA512(password)
	if err != nil {
		return et.Items{}, err
	}

	if user.Password != hash {
		return et.Items{}, errors.New(msg.MSG_INVALID_PASSWORD)
	}

	device := "apiRest"
	duration := time.Minute * 60
	token, err := claim.NewToken(appName, device, user.ID, user.Username, et.Json{
		"database": db.Name,
		"user_id":  user.ID,
	}, duration)
	if err != nil {
		return et.Items{}, err
	}

	_, err = server.newSession(token, db, HTTP, et.Json{})
	if err != nil {
		return et.Items{}, err
	}

	result := et.Items{}
	result.Add(et.Json{
		"token": token,
	})

	return result, nil
}

/**
* JQuery: Executes a query
* @param query et.Json
* @return et.Items, error
**/
func (s *Server) jQuery(query et.Json) (et.Items, error) {
	token := query.Str("token")
	if token == "" {
		return et.Items{}, errors.New(http.StatusText(http.StatusUnauthorized))
	}

	session, err := s.getSession(token)
	if errors.Is(err, ErrorSessionNotFound) {
		return et.Items{}, errors.New(http.StatusText(http.StatusUnauthorized))
	} else if err != nil {
		return et.Items{}, err
	}

	db := session.DB
	result, err := db.JQuery(query)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

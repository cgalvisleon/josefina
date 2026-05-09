package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwt"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/msg"
)

const (
	version = "1.0.0"
)

/**
* NodeParams: Parameters for creating a new Node
**/
type NodeParams struct {
	Port int    `json:"port"`
	Path string `json:"path"`
}

/**
* Node: Runtime instance of the database engine; owns all databases, sessions, and the TCP transport.
**/
type Node struct {
	Version   string              `json:"version"`
	DBS       map[string]*DB      `json:"dbs"`
	Sessions  map[string]*Session `json:"-"`
	muDbs     *sync.RWMutex       `json:"-"`
	muSession *sync.RWMutex       `json:"-"`
	catalog   *DB                 `json:"-"`
	dbs       *Model              `json:"-"`
	users     *Model              `json:"-"`
	sessions  *Model              `json:"-"`
	tcp       *tcp.Node           `json:"-"`
}

var (
	node *Node
)

/**
* Load: Load the node. Accepts an optional TCP port; falls back to $PORT or 1305.
* @param params NodeParams
* @return (*Node, error)
**/
func Load(params NodeParams) (*Node, error) {
	if params.Port == 0 {
		params.Port = envar.GetInt("PORT", 1305)
	}

	node = &Node{
		Version:   version,
		DBS:       make(map[string]*DB, 0),
		Sessions:  make(map[string]*Session, 0),
		muDbs:     &sync.RWMutex{},
		muSession: &sync.RWMutex{},
		tcp:       tcp.NewNode(params.Port),
	}

	var err error
	name := sysDb
	path := params.Path
	if path == "" {
		path = envar.GetStr("DATA_PATH", "./data")
	}
	node.catalog, err = NewDb(path, name)
	if err != nil {
		return nil, err
	}

	// The catalog DB needs its own transaction/error/cache models before any insert can run.
	node.catalog.node = node
	if err := node.catalog.Init(); err != nil {
		return nil, err
	}

	if err := node.load(); err != nil {
		return nil, err
	}

	return node, nil
}

/**
* load: Load the node
* @return error
**/
func (s *Node) load() error {
	if err := s.loadDbs(); err != nil {
		return err
	}

	if err := s.loadUsers(); err != nil {
		return err
	}

	if err := s.loadSessions(); err != nil {
		return err
	}

	return nil
}

/**
* loadDbs: Load the dbs
* @return error
**/
func (s *Node) loadDbs() error {
	var err error
	s.dbs, err = s.catalog.Define(DModel{
		Name:    "dbs",
		IsCore:  true,
		Version: 1,
	})
	if err != nil {
		return err
	}

	if err := s.dbs.Init(); err != nil {
		return err
	}

	s.dbs.ForEachBt(func(idx string, src []byte) (bool, error) {
		var db DB
		if err := json.Unmarshal(src, &db); err != nil {
			return false, err
		}

		if err := db.load(s); err != nil {
			return false, err
		}

		s.muDbs.Lock()
		s.DBS[db.Name] = &db
		s.muDbs.Unlock()
		return true, nil
	}, false, 0, 0)

	return nil
}

/**
* GetDb: Get a database
* @param name string
* @return (*DB, error)
**/
func (s *Node) GetDb(name string) (*DB, error) {
	s.muDbs.RLock()
	result, exists := s.DBS[name]
	s.muDbs.RUnlock()

	if !exists {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
	}

	return result, nil
}

/**
* Autentication: Autenticate a user
* @param token string
* @return (*Session, error)
**/
func (s *Node) Autentication(token string) (*Session, error) {
	claim, err := jwt.Validate(token)
	if err != nil {
		return nil, err
	}

	db, err := s.GetDb(claim.App)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf(`%s:%s`, claim.UserId, claim.App)
	var result *Session
	exists, err := db.GetCache(key, result)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New(msg.MSG_SESSION_NOT_FOUND)
	}

	return result, nil
}

/**
* Start: Starts the TCP listener for this node.
* @return error
**/
func (s *Node) Start() error {
	return s.tcp.Start()
}

/**
* Authenticate: Validates a JWT token and returns the embedded claim.
* @param token string
* @return (*claim.Claim, error)
**/
func (s *Node) Authenticate(token string) (*claim.Claim, error) {
	return jwt.Validate(token)
}

/**
* ListUsers: Returns all registered users.
* @return (et.Items, error)
**/
func (s *Node) ListUsers() (et.Items, error) {
	return From(s.users).All()
}

/**
* CreateDb: Creates a new user database and registers it in the catalog.
* @param name string
* @return (*DB, error)
**/
func (s *Node) CreateDb(name string) (*DB, error) {
	s.muDbs.RLock()
	_, exists := s.DBS[name]
	s.muDbs.RUnlock()
	if exists {
		return nil, fmt.Errorf(msg.MSG_DB_NOT_FOUND)
	}

	path := envar.GetStr("DATA_PATH", "./data")
	db, err := NewDb(path, name)
	if err != nil {
		return nil, err
	}

	if err := db.load(s); err != nil {
		return nil, err
	}

	if err := db.Save(); err != nil {
		return nil, err
	}

	s.muDbs.Lock()
	s.DBS[name] = db
	s.muDbs.Unlock()

	return db, nil
}

/**
* DropDb: Removes a user database from the catalog and in-memory registry.
* @param name string
* @return error
**/
func (s *Node) DropDb(name string) error {
	s.muDbs.Lock()
	_, exists := s.DBS[name]
	if !exists {
		s.muDbs.Unlock()
		return errors.New(msg.MSG_DB_NOT_FOUND)
	}
	delete(s.DBS, name)
	s.muDbs.Unlock()
	return nil
}

/**
* SignInByPassword: Sign in by password
* @param username, password, database, address string, tp TpConnection, payload et.Json, duration time.Duration
* @return (string, error)
**/
func (s *Node) SignInByPassword(username, password, database, address string, tp TpConnection, payload et.Json, duration time.Duration) (string, error) {
	db, err := s.GetDb(database)
	if err != nil {
		return "", err
	}

	user, err := s.GetUser(username, password)
	if err != nil {
		return "", err
	}

	if !user.Ok {
		return "", errors.New(msg.MSG_USER_NOT_FOUND)
	}

	userId := user.Str(INDEX)
	result, err := newSession(db, userId, username, address, tp, payload, duration)
	if err != nil {
		return "", err
	}

	return result, nil
}

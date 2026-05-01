package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwt"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/msg"
)

const (
	version = "1.0.0"
)

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
* Load: Load the node
* @return error
**/
func Load() error {
	port := envar.GetInt("PORT", 1305)
	node = &Node{
		Version:   version,
		DBS:       make(map[string]*DB, 0),
		Sessions:  make(map[string]*Session, 0),
		muDbs:     &sync.RWMutex{},
		muSession: &sync.RWMutex{},
		tcp:       tcp.NewNode(port),
	}

	var err error
	name := "_catalog"
	path := envar.GetStr("DATA_PATH", "./data")
	node.catalog, err = NewDb(path, name)
	if err != nil {
		return err
	}

	if err := node.load(); err != nil {
		return err
	}

	return nil
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
* loadModels: Load the models
* @param db *DB
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

		err := db.load(s)
		if err != nil {
			return false, err
		}
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

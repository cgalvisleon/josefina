package jdb

import (
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/tcp"
)

const (
	version = "1.0.0"
)

/**
* loadDbs: Loads the schemas
* @param db *DB
* @return error
**/
func loadDbs(db *DB) error {
	result, err := db.NewModel("", "schemas", true, 1)
	if err != nil {
		return err
	}

	err = result.Init()
	if err != nil {
		return err
	}

	db.schemas = result

	return nil
}

type Node struct {
	Version   string              `json:"version"`
	DBS       map[string]*DB      `json:"dbs"`
	Sessions  map[string]*Session `json:"-"`
	muDbs     *sync.RWMutex       `json:"-"`
	muSession *sync.RWMutex       `json:"-"`
	dbs       *Model              `json:"-"`
	users     *Model              `json:"-"`
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

	name := "_catalog"
	path := envar.GetStr("DATA_PATH", "./data")
	catalog, err := NewDb(path, name)
	if err != nil {
		return err
	}

	catalog.node = node
	node.muDbs.Lock()
	node.DBS[name] = catalog
	node.muDbs.Unlock()

	node.dbs, err = catalog.NewModel("", "dbs", true, 1)
	if err != nil {
		return err
	}

	node.users, err = catalog.NewModel("", "users", true, 1)
	if err != nil {
		return err
	}

	return nil
}

/**
* loadModels: Load the models
* @param db *DB
* @return error
**/
func (s *Node) loadModels(db *DB) error {
	model, err := db.NewModel("", "dbs", true, 1)
	if err != nil {
		return err
	}

	if _, err = model.DefineAtrib("name", TpText, ""); err != nil {
		return err
	}
	if _, err = model.DefineAtrib("schemas", TpJson, et.Join{}); err != nil {
		return err
	}
	if err = model.DefineIndexes("name", "schemas"); err != nil {
		return err
	}

	s.dbs = model
	err = s.dbs.Init()
	if err != nil {
		return err
	}

	return nil
}

/**
* loadUsers: Load the users
* @param db *DB
* @return error
**/
func (s *Node) loadUsers(db *DB) error {
	var err error
	s.users, err = db.NewModel("", "users", true, 1)
	if err != nil {
		return err
	}

	err = s.users.DefineIndexes("email", "password")
	if err != nil {
		return err
	}

	err = s.users.Init()
	if err != nil {
		return err
	}

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

	if exists {
		return result, nil
	}

	return nil, nil
}

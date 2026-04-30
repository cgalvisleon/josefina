package jdb

import (
	"encoding/json"
	"sync"

	"github.com/cgalvisleon/et/envar"
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
	catalog   *DB                 `json:"-"`
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

	return nil
}

/**
* loadModels: Load the models
* @param db *DB
* @return error
**/
func (s *Node) loadDbs() error {
	var err error
	s.dbs, err = s.catalog.NewModel("", "dbs", true, 1)
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
* loadUsers: Load the users
* @return error
**/
func (s *Node) loadUsers() error {
	var err error
	s.users, err = s.catalog.NewModel("", "users", true, 1)
	if err != nil {
		return err
	}

	err = s.users.DefineIndexes("email", "password")
	if err != nil {
		return err
	}

	if err = s.users.Init(); err != nil {
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

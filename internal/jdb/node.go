package jdb

import (
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
	*tcp.Node
	Version   string              `json:"version"`
	DBS       map[string]*DB      `json:"dbs"`
	Sessions  map[string]*Session `json:"-"`
	muDbs     *sync.RWMutex       `json:"-"`
	muSession *sync.RWMutex       `json:"-"`
	dbs       *Model              `json:"-"`
	users     *Model              `json:"-"`
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
		Node:      tcp.NewNode(port),
		Version:   version,
		DBS:       make(map[string]*DB, 0),
		Sessions:  make(map[string]*Session, 0),
		muDbs:     &sync.RWMutex{},
		muSession: &sync.RWMutex{},
	}

	name := "_catalog"
	path := envar.GetStr("DATA_PATH", "./data")
	catalog, err := NewDb(path, name)
	if err != nil {
		return err
	}

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

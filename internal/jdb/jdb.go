package jdb

import (
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

var node *Node

/**
* Load: Loads the node
* @param port int
* @return *Node
**/
func Load(port int) *Node {
	if node != nil {
		return node
	}

	config, err := getConfig()
	if err != nil {
		return nil
	}

	node = &Node{
		Server:    tcp.NewServer(port),
		app:       appName,
		version:   version,
		isStrict:  config.IsStrict,
		dbs:       make(map[string]*catalog.DB),
		models:    make(map[string]*catalog.Model),
		sessions:  make(map[string]*Session),
		cache:     make(map[string][]byte),
		muDB:      sync.RWMutex{},
		muModel:   sync.RWMutex{},
		muSession: sync.RWMutex{},
		muCache:   sync.RWMutex{},
		lead:      new(Lead),
		follow:    new(Follow),
	}
	node.Mount(node.lead)

	logs.Debug(node.GetMethod().ToString())
	return node
}

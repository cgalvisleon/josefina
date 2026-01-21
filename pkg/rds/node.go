package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type TypeNode string

const (
	Master TypeNode = "master"
	Follow TypeNode = "follow"
)

type Node struct {
	Type    TypeNode        `json:"type"`
	Host    string          `json:"name"`
	Port    int             `json:"port"`
	Version string          `json:"version"`
	Path    string          `json:"path"`
	master  string          `json:"-"`
	dbs     map[string]*DB  `json:"-"`
	nodes   map[string]bool `json:"-"`
}

/**
* newNode
* @param tp TypeNode, version, path string
* @return *Node
**/
func newNode(tp TypeNode, version, path string) *Node {
	port := envar.GetInt("PORT", 4200)
	return &Node{
		Type:    tp,
		Host:    hostName,
		Port:    port,
		Version: version,
		Path:    path,
		dbs:     make(map[string]*DB),
		nodes:   make(map[string]bool),
	}
}

/**
* newDb
* @param name string
* @return *DB, error
**/
func (s *Node) newDb(name string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := s.dbs[name]
	if ok {
		return result, nil
	}

	result = newDb(s.Path, name, s.Version)
	s.dbs[name] = result

	return result, nil
}

/**
* getDb
* @param name string
* @return *DB, error
**/
func (s *Node) getDb(name string) (*DB, error) {
	result, ok := s.dbs[name]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_DB_NOT_FOUND)
	}

	return result, nil
}

/**
* addNode
* @param node string
**/
func (s *Node) addNode(node string) {
	s.nodes[node] = true
}

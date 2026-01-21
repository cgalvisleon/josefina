package rds

import (
	"fmt"
	"net"
	"net/rpc"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type TypeNode string

const (
	MASTER TypeNode = "master"
	FOLLOW TypeNode = "follow"
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
	started bool            `json:"-"`
}

/**
* newNode
* @param tp TypeNode, host string, port int, path, version string
* @return *Node
**/
func newNode(tp TypeNode, host string, port int, path, version string) *Node {
	return &Node{
		Type:    tp,
		Host:    host,
		Port:    port,
		Version: version,
		Path:    path,
		dbs:     make(map[string]*DB),
		nodes:   make(map[string]bool),
	}
}

/**
* start
* @return error
**/
func (s *Node) start() error {
	if s.started {
		return nil
	}

	address := fmt.Sprintf(`:%d`, s.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logs.Fatal(err)
	}

	s.started = true
	logs.Logf("Rpc", "running on %s%s", s.Host, listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			logs.Panic(err)
			continue
		}

		go rpc.ServeConn(conn)
	}
}

/**
* mount
* @param services any
* @return error
**/
func (s *Node) mount(services any) error {
	return rpc.Register(services)
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

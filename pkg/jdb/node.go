package jdb

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

type TpConnection int

const (
	HTTP TpConnection = iota
	WebSocket
	TCP
)

type Status int

const (
	Connected Status = iota
	Disconnected
)

type Session struct {
	CreatedAt time.Time    `json:"created_at"`
	ID        string       `json:"id"`
	Username  string       `json:"username"`
	Address   string       `json:"address"`
	Status    Status       `json:"status"`
	Type      TpConnection `json:"type"`
	Database  string       `json:"database"`
}

type Node struct {
	*tcp.Server
	app       string                    `json:"-"`
	version   string                    `json:"-"`
	isStrict  bool                      `json:"-"`
	started   bool                      `json:"-"`
	sessions  map[string]*Session       `json:"-"`
	models    map[string]*catalog.Model `json:"-"`
	muSession sync.RWMutex              `json:"-"`
	muModel   sync.RWMutex              `json:"-"`
	isDebug   bool                      `json:"-"`
}

/**
* newNode
* @param port int
* @return *Node
**/
func newNode(port int, isStrict bool) *Node {
	result := &Node{
		Server:    tcp.NewServer(port),
		app:       appName,
		version:   version,
		isStrict:  isStrict,
		sessions:  make(map[string]*Session),
		models:    make(map[string]*catalog.Model),
		muSession: sync.RWMutex{},
		muModel:   sync.RWMutex{},
	}

	return result
}

/**
* toJson: Converts the node to a json
* @return et.Json
**/
func (s *Node) toJson() et.Json {
	leader, _ := s.getLeader()
	return et.Json{
		"app":     s.app,
		"address": s.Address(),
		"port":    s.Port(),
		"version": s.version,
		"leader":  leader,
		"peers":   s.Peers,
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

	nodes, err := getNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		s.AddNode(node)
	}

	err = s.Start()
	if err != nil {
		return err
	}

	s.started = true

	return nil
}

/**
* getLeader
* @return string, error
**/
func (n *Node) getLeader() (string, bool) {
	return n.LeaderID()
}

package jdb

import (
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/config"
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

type Client struct {
	Username string       `json:"username"`
	Address  string       `json:"address"`
	Status   Status       `json:"status"`
	Type     TpConnection `json:"type"`
	Database string       `json:"database"`
}

type Node struct {
	*tcp.Server
	app      string                    `json:"-"`
	version  string                    `json:"-"`
	isStrict bool                      `json:"-"`
	models   map[string]*catalog.Model `json:"-"`
	rpcs     map[string]et.Json        `json:"-"`
	turn     int                       `json:"-"`
	started  bool                      `json:"-"`
	clients  map[string]*Client        `json:"-"`
	mu       sync.Mutex                `json:"-"`
	muModel  sync.RWMutex              `json:"-"`
	muClient sync.RWMutex              `json:"-"`
	isDebug  bool                      `json:"-"`
}

/**
* newNode
* @param host string, port int
* @return *Node
**/
func newNode(host string, port int, isStrict bool) *Node {
	result := &Node{
		Server:   tcp.NewServer(port),
		app:      appName,
		version:  version,
		isStrict: isStrict,
		models:   make(map[string]*catalog.Model),
		rpcs:     make(map[string]et.Json),
		clients:  make(map[string]*Client),
		mu:       sync.Mutex{},
		muModel:  sync.RWMutex{},
		muClient: sync.RWMutex{},
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
		"rpcs":    s.rpcs,
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

	nodes, err := config.GetNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		s.AddNode(node)
	}

	err = jrpc.Start(s.port)
	if err != nil {
		return err
	}

	go s.ElectionLoop()

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

/**
* reportModels: Reports the models
* @param models map[string]*catalog.Model
* @return error
**/
func (s *Node) reportModels(models map[string]*catalog.Model) error {
	leader, imLeader := s.getLeader()
	if !imLeader {
		var response bool
		err := jrpc.Call(leader, "Node.ReportModels", models, &response)
		if err != nil {
			return err
		}

		return nil
	}

	for key, model := range models {
		s.muModel.Lock()
		s.models[key] = model
		s.muModel.Unlock()
	}

	return nil
}

/**
* ReportModels: Reports models
* @param require map[string]*Model, response true
* @return error
**/
func (s *Node) ReportModels(require map[string]*catalog.Model, response *bool) error {
	err := s.reportModels(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* onConnect: Sets the client
* @param username string, tpConnection TpConnection, address string
**/
func (s *Node) onConnect(username string, tpConnection TpConnection, address string) error {
	leader, imLeader := s.getLeader()
	if !imLeader {
		args := et.Json{
			"username":     username,
			"tpConnection": tpConnection,
			"address":      address,
		}
		var dest bool
		err := jrpc.Call(leader, "Node.OnConnect", args, &dest)
		if err != nil {
			return err
		}

		return nil
	}

	s.muClient.Lock()
	s.clients[username] = &Client{
		Username: username,
		Address:  address,
		Type:     tpConnection,
		Status:   Connected,
	}
	s.muClient.Unlock()

	return nil
}

/**
* OnConnect: Handles a connection
* @param require et.Json, response *boolean
* @return error
**/
func (s *Node) OnConnect(require et.Json, response *bool) error {
	username := require.Str("username")
	tpConnection := TpConnection(require.Int("tpConnection"))
	address := require.Str("address")
	err := s.onConnect(username, tpConnection, address)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* onDisconnect: Removes the client
* @param username string
**/
func (s *Node) onDisconnect(username string) error {
	leader, imLeader := s.getLeader()
	if !imLeader {
		var dest bool
		err := jrpc.Call(leader, "Node.OnDisconnect", username, &dest)
		if err != nil {
			return err
		}

		return nil
	}

	s.muClient.Lock()
	delete(s.clients, username)
	s.muClient.Unlock()
	return nil
}

/**
* OnDisconnect: Handles a disconnection
* @param require string, response *boolean
* @return error
**/
func (s *Node) OnDisconnect(require string, response *bool) error {
	err := s.onDisconnect(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

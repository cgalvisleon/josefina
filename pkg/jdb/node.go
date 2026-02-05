package jdb

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/config"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
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
	PackageName   string                    `json:"packageName"`
	Version       string                    `json:"version"`
	Address       string                    `json:"address"`
	port          int                       `json:"-"`
	isStrict      bool                      `json:"-"`
	models        map[string]*catalog.Model `json:"-"`
	rpcs          map[string]et.Json        `json:"-"`
	peers         []string                  `json:"-"`
	state         NodeState                 `json:"-"`
	term          int                       `json:"-"`
	votedFor      string                    `json:"-"`
	leaderID      string                    `json:"-"`
	lastHeartbeat time.Time                 `json:"-"`
	turn          int                       `json:"-"`
	started       bool                      `json:"-"`
	clients       map[string]*Client        `json:"-"`
	mu            sync.Mutex                `json:"-"`
	modelMu       sync.RWMutex              `json:"-"`
	clientMu      sync.RWMutex              `json:"-"`
	isDebug       bool                      `json:"-"`
}

/**
* newNode
* @param host string, port int
* @return *Node
**/
func newNode(host string, port int, isStrict bool) *Node {
	address := fmt.Sprintf(`%s:%d`, host, port)
	result := &Node{
		PackageName: appName,
		Address:     address,
		Version:     version,
		port:        port,
		isStrict:    isStrict,
		models:      make(map[string]*catalog.Model),
		rpcs:        make(map[string]et.Json),
		clients:     make(map[string]*Client),
		mu:          sync.Mutex{},
		modelMu:     sync.RWMutex{},
		clientMu:    sync.RWMutex{},
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
		"address": s.Address,
		"leader":  leader,
		"version": s.Version,
		"rpcs":    s.rpcs,
		"peers":   s.peers,
	}
}

/**
* mount: Mounts the services
* @param services any
* @return error
**/
func (s *Node) mount(services any) error {
	router, err := jrpc.Mount(s.Address, services)
	if err != nil {
		return err
	}

	for name, rpc := range router {
		s.rpcs[name] = rpc
	}

	return nil
}

/**
* addPeer
* @param node string
**/
func (s *Node) addPeer(node string) {
	s.peers = append(s.peers, node)
}

/**
* next
* @return string
**/
func (s *Node) next() string {
	t := len(s.peers)
	if t == 0 {
		return s.Address
	}

	s.turn++
	if s.turn >= t {
		s.turn = 1
	}

	return s.peers[s.turn]
}

/**
* getLeader
* @return string, error
**/
func (n *Node) getLeader() (string, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	result := n.leaderID
	return result, result != n.Address && result != ""
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
		s.addPeer(node)
	}

	err = jrpc.Start(s.port)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.state = Follower
	s.lastHeartbeat = timezone.Now()
	s.mu.Unlock()
	go s.electionLoop()

	s.started = true

	return nil
}

/**
* Ping
* @param to string
* @return bool
**/
func (s *Node) Ping(to string) bool {
	var response string
	err := jrpc.CallRpc(to, "Node.Pong", s.Address, &response)
	if err != nil {
		return false
	}

	logs.Logf(s.PackageName, "%s:%s", response, to)
	return true
}

/**
* Pong: Pings the leader
* @param response *string
* @return error
**/
func (s *Node) Pong(require string, response *string) error {
	logs.Log(s.PackageName, "ping:", require)
	*response = "pong"
	return nil
}

/**
* loadModel: Loads a model
* @param to string, model *Model
* @return (*Model, error)
**/
func (s *Node) loadModel(to string, model *catalog.Model) (*catalog.Model, error) {
	var response *catalog.Model
	err := jrpc.CallRpc(to, "Sync.LoadModel", model, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* GetModel: Gets a model
* @param require *catalog.From, response *catalog.Model
* @return error
**/
func (s *Node) GetModel(require *catalog.From, response *catalog.Model) error {
	key := require.Key()
	s.modelMu.RLock()
	result, ok := s.models[key]
	s.modelMu.RUnlock()
	if ok {
		response = result
		return nil
	}

	exists, err := core.GetModel(require, response)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if !response.IsInit {
		host := s.next()
		response, err = s.loadModel(host, response)
		if err != nil {
			return err
		}
	}

	s.modelMu.Lock()
	s.models[key] = response
	s.modelMu.Unlock()

	return nil
}

/**
* reportModels: Reports the models
* @param models map[string]*catalog.Model
* @return error
**/
func (s *Node) reportModels(models map[string]*catalog.Model) error {
	leader, ok := s.getLeader()
	if ok {
		var response bool
		err := jrpc.CallRpc(leader, "Node.ReportModels", models, &response)
		if err != nil {
			return err
		}

		return nil
	}

	for key, model := range models {
		s.modelMu.Lock()
		s.models[key] = model
		s.modelMu.Unlock()
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
	leader, ok := s.getLeader()
	if ok {
		args := et.Json{
			"username":     username,
			"tpConnection": tpConnection,
			"address":      address,
		}
		var dest bool
		err := jrpc.CallRpc(leader, "Node.OnConnect", args, &dest)
		if err != nil {
			return err
		}

		return nil
	}

	s.clientMu.Lock()
	s.clients[username] = &Client{
		Username: username,
		Address:  address,
		Type:     tpConnection,
		Status:   Connected,
	}
	s.clientMu.Unlock()

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
	leader, ok := s.getLeader()
	if ok {
		var dest bool
		err := jrpc.CallRpc(leader, "Node.OnDisconnect", username, &dest)
		if err != nil {
			return err
		}

		return nil
	}

	s.clientMu.Lock()
	delete(s.clients, username)
	s.clientMu.Unlock()
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

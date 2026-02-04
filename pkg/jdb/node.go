package jdb

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/ws"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/mod"
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
	Host     string       `json:"host"`
	Status   Status       `json:"status"`
	Type     TpConnection `json:"type"`
}

type Node struct {
	PackageName   string                `json:"packageName"`
	Version       string                `json:"version"`
	Address       string                `json:"address"`
	Port          int                   `json:"port"`
	isStrict      bool                  `json:"-"`
	models        map[string]*mod.Model `json:"-"`
	rpcs          map[string]et.Json    `json:"-"`
	peers         []string              `json:"-"`
	state         NodeState             `json:"-"`
	term          int                   `json:"-"`
	votedFor      string                `json:"-"`
	leaderID      string                `json:"-"`
	lastHeartbeat time.Time             `json:"-"`
	turn          int                   `json:"-"`
	started       bool                  `json:"-"`
	ws            *ws.Hub               `json:"-"`
	clients       map[string]*Client    `json:"-"`
	mu            sync.Mutex            `json:"-"`
	modelMu       sync.RWMutex          `json:"-"`
	clientMu      sync.RWMutex          `json:"-"`
	isDebug       bool                  `json:"-"`
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
		Port:        port,
		Version:     version,
		isStrict:    isStrict,
		models:      make(map[string]*mod.Model),
		rpcs:        make(map[string]et.Json),
		ws:          ws.NewWs(),
		clients:     make(map[string]*Client),
		mu:          sync.Mutex{},
		modelMu:     sync.RWMutex{},
		clientMu:    sync.RWMutex{},
	}
	result.ws.OnConnection(func(subscriber *ws.Subscriber) {
		result.onConnect(subscriber.Name, WebSocket, result.Address)
	})
	result.ws.OnDisconnection(func(subscriber *ws.Subscriber) {
		result.onDisconnect(subscriber.Name)
	})

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
* setDebug
* @param debug bool
**/
func (s *Node) setDebug(debug bool) {
	s.isDebug = debug
}

/**
* addNode
* @param node string
**/
func (s *Node) addNode(node string) {
	s.peers = append(s.peers, node)
}

/**
* nextHost
* @return string
**/
func (s *Node) nextHost() string {
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

	nodes, err := getNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		s.addNode(node)
	}

	err = jrpc.Start(s.Port)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.state = Follower
	s.lastHeartbeat = timezone.Now()
	s.mu.Unlock()
	s.ws.Start()
	s.ws.SetDebug(s.isDebug)
	go s.electionLoop()
	s.started = true

	return nil
}

/**
* ping
* @param to string
* @return bool
**/
func (s *Node) ping(to string) bool {
	var response string
	err := jrpc.CallRpc(to, "Node.Ping", s.Address, &response)
	if err != nil {
		return false
	}

	logs.Logf(s.PackageName, "%s:%s", response, to)
	return true
}

/**
* Ping: Pings the leader
* @param response *string
* @return error
**/
func (s *Node) Ping(require string, response *string) error {
	logs.Log(s.PackageName, "ping:", require)
	*response = "pong"
	return nil
}

/**
* loadModel: Loads a model
* @param to string, model *Model
* @return (*Model, error)
**/
func (s *Node) loadModel(to string, model *mod.Model) (*mod.Model, error) {
	var response *mod.Model
	err := jrpc.CallRpc(to, "Mod.LoadModel", model, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* GetModel: Gets a model
* @param require *mod.From, response *mod.Model
* @return error
**/
func (s *Node) GetModel(require *mod.From, response *mod.Model) error {
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
		host := node.nextHost()
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
* @param models map[string]*mod.Model
* @return error
**/
func (s *Node) reportModels(models map[string]*mod.Model) error {
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
		s.mu.Lock()
		s.models[key] = model
		s.mu.Unlock()
	}

	return nil
}

/**
* ReportModels: Reports models
* @param require map[string]*Model, response true
* @return error
**/
func (s *Node) ReportModels(require map[string]*mod.Model, response *bool) error {
	err := s.reportModels(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* authenticate: Authenticates a user
* @param token string
* @return *claim.Claim, error
**/
func (s *Node) authenticate(token string) (*claim.Claim, error) {
	leader, ok := s.getLeader()
	if ok {
		var reply *claim.Claim
		err := jrpc.CallRpc(leader, "Node.Authenticate", token, &reply)
		if err != nil {
			return nil, err
		}

		return reply, nil
	}

	return core.Authenticate(token)
}

/**
* Auth: Authenticates a user
* @param require string, response *Claim
* @return error
**/
func (s *Node) Authenticate(require string, response *claim.Claim) error {
	result, err := s.authenticate(require)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* auth
* @param device, database, username, password string
* @return *Session, error
**/
func (s *Node) auth(device, database, username, password string) (*core.Session, error) {
	leader, ok := s.getLeader()
	if ok {
		args := et.Json{
			"device":   device,
			"database": database,
			"username": username,
			"password": password,
		}
		var reply *core.Session
		err := jrpc.CallRpc(leader, "Node.Auth", args, &reply)
		if err != nil {
			return nil, err
		}

		return reply, nil
	}

	item, err := core.GetUser(username, password)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, errors.New(msg.MSG_AUTHENTICATION_FAILED)
	}

	result, err := core.CreateSession(device, username)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s:%s:%s", appName, device, username)
	_, err = cache.Set(key, result.Token, 0)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Auth: Authenticates a user
* @param require et.Json, response *Session
* @return error
**/
func (s *Node) Auth(require et.Json, response *core.Session) error {
	device := require.Str("device")
	database := require.Str("database")
	username := require.Str("username")
	password := require.Str("password")
	result, err := s.auth(device, database, username, password)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* onConnect: Sets the client
* @param username string, tpConnection TpConnection, host string
**/
func (s *Node) onConnect(username string, tpConnection TpConnection, host string) error {
	leader, ok := s.getLeader()
	if ok {
		args := et.Json{
			"username":     username,
			"tpConnection": tpConnection,
			"host":         host,
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
		Host:     host,
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
	host := require.Str("host")
	err := s.onConnect(username, tpConnection, host)
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

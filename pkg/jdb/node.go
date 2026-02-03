package jdb

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
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
* ToJson: Converts the node to a json
* @return et.Json
**/
func (s *Node) ToJson() et.Json {
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
* helpCheck: Returns the help check
* @return et.Json
**/
func (s *Node) helpCheck() et.Json {
	return et.Json{
		"address": s.Address,
		"leader":  s.leaderID,
		"version": s.Version,
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
* SetDebug
* @param debug bool
**/
func (s *Node) SetDebug(debug bool) {
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

	err := s.mount(syn)
	if err != nil {
		return err
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
* Ping
* @param to string
* @return bool
**/
func (s *Node) Ping(to string) bool {
	err := syn.ping(to)
	if err != nil {
		return false
	}

	return true
}

/**
* reportModels: Reports the models
* @param models map[string]*mod.Model
* @return error
**/
func (s *Node) reportModels(models map[string]*mod.Model) error {
	leader, ok := s.getLeader()
	if ok {
		return syn.reportModels(leader, models)
	}

	for key, model := range models {
		s.mu.Lock()
		s.models[key] = model
		s.mu.Unlock()
	}

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
		return syn.authenticate(leader, token)
	}

	return core.Authenticate(token)
}

/**
* auth
* @param device, database, username, password string
* @return *Session, error
**/
func (s *Node) auth(device, database, username, password string) (*core.Session, error) {
	leader, ok := s.getLeader()
	if ok {
		return syn.auth(leader, device, database, username, password)
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
* onConnect: Sets the client
* @param username string, tpConnection TpConnection, host string
**/
func (s *Node) onConnect(username string, tpConnection TpConnection, host string) error {
	leader, ok := s.getLeader()
	if ok {
		return syn.onConnect(leader, username, tpConnection, host)
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
* onDisconnect: Removes the client
* @param username string
**/
func (s *Node) onDisconnect(username string) error {
	leader, ok := s.getLeader()
	if ok {
		return syn.onDisconnect(leader, username)
	}

	s.clientMu.Lock()
	delete(s.clients, username)
	s.clientMu.Unlock()
	return nil
}

package node

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/dbs"
	"github.com/cgalvisleon/josefina/pkg/msg"
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
	Host          string                `json:"host"`
	Port          int                   `json:"port"`
	isStrict      bool                  `json:"-"`
	dbs           map[string]*dbs.DB    `json:"-"`
	models        map[string]*dbs.Model `json:"-"`
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

var (
	packageName string = "josefina"
	version     string = "0.0.1"
	// node        *Node
)

func init() {
	// hostname, err := os.Hostname()
	// if err != nil {
	// 	hostname = "localhost"
	// }

	// port := envar.GetInt("RPC_PORT", 4200)
	// node = newNode(hostname, port)
}

/**
* newNode
* @param host string, port int
* @return *Node
**/
func newNode(host string, port int, isStrict bool) *Node {
	address := fmt.Sprintf(`%s:%d`, host, port)
	result := &Node{
		PackageName: packageName,
		Host:        address,
		Port:        port,
		Version:     version,
		isStrict:    isStrict,
		rpcs:        make(map[string]et.Json),
		dbs:         make(map[string]*dbs.DB),
		models:      make(map[string]*dbs.Model),
		ws:          ws.NewWs(),
		clients:     make(map[string]*Client),
		mu:          sync.Mutex{},
		modelMu:     sync.RWMutex{},
		clientMu:    sync.RWMutex{},
	}
	result.ws.OnConnection(func(subscriber *ws.Subscriber) {
		result.onConnect(subscriber.Name, WebSocket, result.Host)
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
		"host":    s.Host,
		"leader":  leader,
		"version": s.Version,
		"rpcs":    s.rpcs,
		"peers":   s.peers,
		"models":  s.models,
	}
}

/**
* helpCheck: Returns the help check
* @return et.Json
**/
func (s *Node) helpCheck() et.Json {
	return et.Json{
		"host":    s.Host,
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
	router, err := jrpc.Mount(s.Host, services)
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
		return s.Host
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
	return result, result != n.Host && result != ""
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
	s.started = true
	s.mu.Unlock()
	s.ws.Start()
	s.ws.SetDebug(s.isDebug)

	go s.electionLoop()

	return nil
}

/**
* ping
* @param to string
* @return bool
**/
func (s *Node) ping(to string) bool {
	err := syn.ping(to)
	if err != nil {
		return false
	}

	return true
}

/**
* setIsStrict
* @param isStrict bool
**/
func (s *Node) setIsStrict(isStrict bool) {
	s.isStrict = isStrict
}

/**
* getDb: Returns a database by name
* @param name string
* @return *DB, error
**/
func (s *Node) getDb(name string) (*dbs.DB, error) {
	if !s.started {
		return nil, errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	leader, ok := s.getLeader()
	if ok {
		return syn.getDb(leader, name)
	}

	name = utility.Normalize(name)
	result, ok := s.dbs[name]
	if ok {
		return result, nil
	}

	exists, err := core.GetDb(name, result)
	if err != nil {
		return nil, err
	}

	if exists {
		result.SetDebug(s.isDebug)
		return result, nil
	}

	if s.isStrict {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
	}

	result, err = dbs.GetDb(name)
	if err != nil {
		return nil, err
	}

	err = core.SetDb(result)
	if err != nil {
		return nil, err
	}

	s.dbs[name] = result

	return result, nil
}

/**
* saveDb: Saves the model
* @param db *DB
* @return error
**/
func (s *Node) saveDb(db *dbs.DB) error {
	if !s.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	leader, ok := s.getLeader()
	if ok {
		return syn.saveDb(leader, db)
	}

	return core.SetDb(db)
}

/**
* getModel
* @param database, schema, name string
* @return *dbs.Model, error
**/
func (s *Node) getModel(database, schema, name string) (*dbs.Model, error) {
	if !s.started {
		return nil, errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(database, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	loadModel := func(result *dbs.Model) (*dbs.Model, error) {
		to := s.nextHost()
		if to == s.Host {
			err := s.loadModel(result)
			if err != nil {
				return nil, err
			}
		} else {
			err := syn.loadModel(to, result)
			if err != nil {
				return nil, err
			}
		}

		return result, nil
	}

	leader, ok := s.getLeader()
	if ok {
		return syn.getModel(leader, database, schema, name)
	}

	key := modelKey(database, schema, name)
	s.modelMu.RLock()
	result, ok := s.models[key]
	s.modelMu.RUnlock()
	if ok {
		return result, nil
	}

	exists, err := core.GetModel(&dbs.From{
		Database: database,
		Schema:   schema,
		Name:     name,
	}, result)
	if err != nil {
		return nil, err
	}

	if exists {
		if result.IsInit {
			return result, nil
		}

		result, err = loadModel(result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	db, err := s.getDb(database)
	if err != nil {
		return nil, err
	}

	if db.IsStrict {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	result, err = db.NewModel(schema, name, false, 1)
	if err != nil {
		return nil, err
	}

	err = core.SetModel(result)
	if err != nil {
		return nil, err
	}

	result, err = loadModel(result)
	if err != nil {
		return nil, err
	}

	s.modelMu.Lock()
	s.models[key] = result
	s.modelMu.Unlock()

	return result, nil
}

/**
* reserveModel
* @param model *Model
* @return error
**/
func (s *Node) loadModel(model *dbs.Model) error {
	if !s.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	err := model.Init()
	if err != nil {
		return err
	}

	key := model.Key()
	s.modelMu.Lock()
	s.models[key] = model
	s.modelMu.Unlock()

	return nil
}

/**
* saveModel: Saves the model
* @param model *Model
* @return error
**/
func (s *Node) saveModel(model *dbs.Model) error {
	if !s.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if model.IsCore {
		return nil
	}

	leader, ok := s.getLeader()
	if ok {
		return syn.saveModel(leader, model)
	}

	return core.SetModel(model)
}

/**
* reportModels: Reports the models
* @param models map[string]*dbs.Model
* @return error
**/
func (s *Node) reportModels(models map[string]*dbs.Model) error {
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
* onConnect: Sets the client
* @param username string
* @param tpConnection TpConnection
* @param host string
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

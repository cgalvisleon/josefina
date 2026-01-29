package jdb

import (
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/cgalvisleon/josefina/pkg/ws"
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

type Client struct {
	Username string       `json:"username"`
	Host     string       `json:"host"`
	Status   Status       `json:"status"`
	Type     TpConnection `json:"type"`
}

type Node struct {
	host          string             `json:"-"`
	port          int                `json:"-"`
	version       string             `json:"-"`
	rpcs          map[string]et.Json `json:"-"`
	dbs           map[string]*DB     `json:"-"`
	models        map[string]*Model  `json:"-"`
	peers         []string           `json:"-"`
	state         NodeState          `json:"-"`
	term          int                `json:"-"`
	votedFor      string             `json:"-"`
	leaderID      string             `json:"-"`
	lastHeartbeat time.Time          `json:"-"`
	turn          int                `json:"-"`
	started       bool               `json:"-"`
	ws            *ws.Hub            `json:"-"`
	clients       map[string]*Client `json:"-"`
	mu            sync.Mutex         `json:"-"`
	modelMu       sync.RWMutex       `json:"-"`
}

/**
* newNode
* @param host string, port int, version string
* @return *Node
**/
func newNode(host string, port int, version string) *Node {
	address := fmt.Sprintf(`%s:%d`, host, port)
	result := &Node{
		host:    address,
		port:    port,
		version: version,
		rpcs:    make(map[string]et.Json),
		dbs:     make(map[string]*DB),
		models:  make(map[string]*Model),
		ws:      ws.NewWs(),
		clients: make(map[string]*Client),
		mu:      sync.Mutex{},
		modelMu: sync.RWMutex{},
	}
	result.ws.OnConnection(func(subscriber *ws.Subscriber) {
		result.onConnect(subscriber.Name, WebSocket, result.host)
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
	leader := s.getLeader()
	return et.Json{
		"host":    s.host,
		"leader":  leader,
		"version": s.version,
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
		"host":    s.host,
		"leader":  s.leaderID,
		"version": s.version,
		"peers":   s.peers,
	}
}

/**
* mount
* @param services any
* @return error
**/
func (s *Node) mount(services any) error {
	tipoStruct := reflect.TypeOf(services)
	structName := tipoStruct.String()
	list := strings.Split(structName, ".")
	structName = list[len(list)-1]
	for i := 0; i < tipoStruct.NumMethod(); i++ {
		metodo := tipoStruct.Method(i)
		numInputs := metodo.Type.NumIn()
		numOutputs := metodo.Type.NumOut()

		inputs := []string{}
		for i := 1; i < numInputs; i++ {
			paramType := metodo.Type.In(i)
			inputs = append(inputs, paramType.String())
		}

		outputs := []string{}
		for i := 0; i < numOutputs; i++ {
			paramType := metodo.Type.Out(i)
			outputs = append(outputs, paramType.String())
		}

		name := fmt.Sprintf("%s.%s", structName, metodo.Name)
		s.rpcs[name] = et.Json{
			"inputs":  inputs,
			"outputs": outputs,
		}

		logs.Logf("rpc", "RPC:/%s/%s", s.host, name)
	}

	return rpc.Register(services)
}

/**
* addNode
* @param node string
**/
func (s *Node) addNode(node string) {
	s.peers = append(s.peers, node)
}

/**
* nextNode
* @return string
**/
func (s *Node) nextNode() string {
	t := len(s.peers)
	if t == 0 {
		return s.host
	}

	s.turn++
	if s.turn >= t {
		s.turn = 1
	}

	return s.peers[s.turn]
}

/**
* getLeader
* @return string
**/
func (n *Node) getLeader() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.leaderID
}

/**
* startRPC
* @param listener net.Listener
**/
func (n *Node) startRPC(listener net.Listener) {
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				logs.Error(err)
				continue
			}

			go rpc.ServeConn(conn)
		}
	}()
}

/**
* start
* @return error
**/
func (s *Node) start() error {
	if s.started {
		return nil
	}

	if methods == nil {
		methods = new(Methods)
	}

	err := s.mount(methods)
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

	address := fmt.Sprintf(`:%d`, s.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logs.Fatal(err)
	}

	s.startRPC(listener)

	s.mu.Lock()
	s.state = Follower
	s.lastHeartbeat = timezone.Now()
	s.started = true
	s.mu.Unlock()
	s.ws.Start()

	go s.electionLoop()

	logs.Logf("Rpc", "running on %s%s", s.host, listener.Addr())
	return nil
}

/**
* Ping
* @param to string
* @return bool
**/
func (s *Node) Ping(to string) bool {
	err := methods.ping(to)
	if err != nil {
		return false
	}

	return true
}

/**
* getModel
* @param database, schema, name string
* @return *From, error
**/
func (s *Node) getModel(database, schema, name string) (*Model, error) {
	if !s.started {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(database, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	leader := s.getLeader()
	if leader != s.host && leader != "" {
		result, err := methods.getModel(leader, database, schema, name)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	type Result struct {
		result *Model
		err    error
	}

	ch := make(chan Result)
	go func() {
		key := modelKey(database, schema, name)
		s.modelMu.RLock()
		result, ok := s.models[key]
		s.modelMu.RUnlock()
		if ok {
			ch <- Result{result: result, err: nil}
			return
		}

		err := initModels()
		if err != nil {
			ch <- Result{result: nil, err: err}
			return
		}

		exists, err := models.get(key, &result)
		if err != nil {
			ch <- Result{result: nil, err: err}
			return
		}

		if !exists {
			ch <- Result{result: nil, err: fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)}
			return
		}

		to := s.nextNode()
		err = methods.loadModel(to, result)
		if err != nil {
			ch <- Result{result: nil, err: err}
			return
		}

		result.Host = to
		result.IsInit = true

		s.modelMu.Lock()
		s.models[key] = result
		s.modelMu.Unlock()
		ch <- Result{result: result, err: nil}
	}()

	res := <-ch
	return res.result, res.err
}

/**
* reserveModel
* @param model *Model
* @return error
**/
func (s *Node) loadModel(model *Model) error {
	if !s.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}

	ch := make(chan error)
	go func() {
		err := model.init()
		if err != nil {
			ch <- err
			return
		}

		key := model.key()
		s.modelMu.RLock()
		result, ok := s.models[key]
		s.modelMu.RUnlock()
		if !ok {
			ch <- fmt.Errorf(msg.MSG_GET_FROM_NOT_USED)
			return
		}

		s.modelMu.Lock()
		s.models[key] = result
		s.modelMu.Unlock()
		ch <- nil
	}()

	res := <-ch
	return res
}

/**
* saveModel: Saves the model
* @param model *Model
* @return error
**/
func (s *Node) saveModel(model *Model) error {
	if !s.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if model.IsCore {
		return nil
	}

	leader := s.getLeader()
	if leader != s.host && leader != "" {
		err := methods.saveModel(leader, model)
		if err != nil {
			return err
		}

		return nil
	}

	ch := make(chan error)
	go func() {
		err := initModels()
		if err != nil {
			ch <- err
			return
		}

		bt, err := model.serialize()
		if err != nil {
			ch <- err
			return
		}

		key := model.key()
		err = models.put(key, bt)
		if err != nil {
			ch <- err
			return
		}

		ch <- nil
	}()

	res := <-ch
	return res
}

/**
* reportModels: Reports the models
* @param models map[string]*Model
* @return error
**/
func (s *Node) reportModels(models map[string]*Model) error {
	leader := s.getLeader()
	if leader != s.host && leader != "" {
		err := methods.reportModels(leader, models)
		if err != nil {
			return err
		}

		return nil
	}

	ch := make(chan error)
	go func() {
		for key, model := range models {
			s.mu.Lock()
			s.models[key] = model
			s.mu.Unlock()
		}
		ch <- nil
	}()

	return <-ch
}

/**
* saveDb: Saves the model
* @param db *DB
* @return error
**/
func (s *Node) saveDb(db *DB) error {
	if !s.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}

	leader := s.getLeader()
	if leader != s.host && leader != "" {
		err := methods.saveDb(leader, db)
		if err != nil {
			return err
		}

		return nil
	}

	ch := make(chan error)
	go func() {
		err := initDbs()
		if err != nil {
			ch <- err
			return
		}

		bt, err := db.serialize()
		if err != nil {
			ch <- err
			return
		}

		key := db.Name
		err = dbs.put(key, bt)
		if err != nil {
			ch <- err
			return
		}

		ch <- nil
	}()

	res := <-ch
	return res
}

/**
* onConnect: Sets the client
* @param username string
* @param tpConnection TpConnection
* @param host string
**/
func (s *Node) onConnect(username string, tpConnection TpConnection, host string) error {
	leader := s.getLeader()
	if leader != s.host && leader != "" {
		return methods.onConnect(leader, username, tpConnection, host)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[username] = &Client{
		Username: username,
		Host:     host,
		Type:     tpConnection,
		Status:   Connected,
	}

	return nil
}

/**
* onDisconnect: Removes the client
* @param username string
**/
func (s *Node) onDisconnect(username string) error {
	leader := s.getLeader()
	if leader != s.host && leader != "" {
		return methods.onDisconnect(leader, username)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, username)
	return nil
}

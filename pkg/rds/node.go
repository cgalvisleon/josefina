package rds

import (
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Node struct {
	host    string             `json:"-"`
	port    int                `json:"-"`
	version string             `json:"-"`
	rpcs    map[string]et.Json `json:"-"`
	dbs     map[string]*DB     `json:"-"`
	models  map[string]*Model  `json:"-"`
	started bool               `json:"-"`
	mu      sync.Mutex         `json:"-"`
}

/**
* newNode
* @param host string, port int, version string
* @return *Node
**/
func newNode(host string, port int, version string) *Node {
	address := fmt.Sprintf(`%s:%d`, host, port)
	return &Node{
		host:    address,
		port:    port,
		version: version,
		rpcs:    make(map[string]et.Json),
		dbs:     make(map[string]*DB),
		models:  make(map[string]*Model),
		mu:      sync.Mutex{},
	}
}

/**
* leader
* @return (string, bool, error)
**/
func (s *Node) leader() (string, bool, error) {
	if methods == nil {
		return "", false, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	config, err := getConfig()
	if err != nil {
		return "", false, err
	}

	t := len(config.Nodes)
	if t == 0 {
		return s.host, false, nil
	}

	leader := config.Leader
	if leader >= t {
		leader = 0
	}

	for {
		if leader >= t {
			break
		}

		result := config.Nodes[leader]
		ok := s.ping(result)
		if ok {
			config.Leader = leader
			err = writeConfig(config)
			if err != nil {
				return "", false, err
			}

			return result, true, nil
		}

		leader++
	}

	return "", false, fmt.Errorf(msg.MSG_NO_LEADER_FOUND)
}

/**
* ping
* @param to string
* @return bool
**/
func (s *Node) ping(to string) bool {
	err := methods.ping(to)
	if err != nil {
		return false
	}

	return true
}

/**
* toJson: Converts the node to a json
* @return et.Json
**/
func (s *Node) toJson() et.Json {
	leader, cluster, err := s.leader()
	if err != nil {
		leader = err.Error()
	}
	return et.Json{
		"host":    s.host,
		"leader":  leader,
		"cluster": cluster,
		"version": s.version,
		"rpcs":    s.rpcs,
		"models":  s.models,
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

	address := fmt.Sprintf(`:%d`, s.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logs.Fatal(err)
	}

	s.started = true
	logs.Logf("Rpc", "running on %s%s", s.host, listener.Addr())

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
* getModel
* @param database, schema, name string
* @return *Model, error
**/
func (s *Node) getModel(database, schema, name string) (*Model, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getModelRetry(database, schema, name, false)
}

/**
* getModelRetry
* @param database, schema, name string
* @return *Model, error
**/
func (s *Node) getModelRetry(database, schema, name string, retried bool) (*Model, error) {
	if !s.started {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(database, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	leader, isCluster, err := s.leader()
	if err != nil {
		return nil, err
	}

	if leader != s.host {
		result, err := methods.getModel(leader, database, schema, name)
		if err != nil {
			return nil, err
		}

		loaded, err := s.loadModel(result)
		if err != nil {
			return nil, err
		}
		if !loaded {
			if retried {
				return nil, fmt.Errorf(msg.MSG_MODEL_NOT_LOAD)
			}
			return s.getModelRetry(database, schema, name, true)
		}

		return result, nil
	}

	key := modelKey(database, schema, name)
	result, ok := s.models[key]
	if ok {
		return result, nil
	}

	err = initModels()
	if err != nil {
		return nil, err
	}

	exists, err := models.get(key, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	if !isCluster {
		err = result.init()
		if err != nil {
			return nil, err
		}
	}

	s.models[key] = result
	return result, nil
}

/**
* loadModel
* @param model *Model
* @return error
**/
func (s *Node) loadModel(model *Model) (bool, error) {
	if !s.started {
		return false, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}

	if model.IsInit {
		return true, nil
	}

	model.IsInit = true
	model.Host = s.host
	ok, err := s.reserveModel(model)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, nil
	}

	err = model.load()
	if err != nil {
		return false, err
	}

	s.models[model.key()] = model

	return true, nil
}

/**
* reserveModel: Saves the model
* @param model *Model
* @return error
**/
func (s *Node) reserveModel(model *Model) (bool, error) {
	if !s.started {
		return false, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}

	if model.IsCore {
		return false, nil
	}

	leader, _, err := s.leader()
	if err != nil {
		return false, err
	}

	if leader != s.host {
		ok, err := methods.reserveModel(leader, model)
		if err != nil {
			return false, err
		}

		return ok, nil
	}

	key := model.key()
	result, ok := s.models[key]
	if !ok {
		s.models[key] = model
		return true, nil
	}

	if result.Host == "" {
		result.Host = model.Host
		result.IsInit = model.IsInit
		s.models[key] = result
		return true, nil
	}

	return false, nil
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

	leader, _, err := s.leader()
	if err != nil {
		return err
	}

	if leader != s.host {
		ok, err := methods.saveModel(leader, model)
		if err != nil {
			return err
		}

		return nil
	}

	err = initModels()
	if err != nil {
		return err
	}

	src, err := model.serialize()
	if err != nil {
		return err
	}

	key := modelKey(model.Database, model.Schema, model.Name)
	err = models.put(key, src)
	if err != nil {
		return err
	}

	return nil
}

/**
* signIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func (s *Node) signIn(device, database, username, password string) (*Session, error) {
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_PASSWORD_REQUIRED)
	}

	if s.leader != s.host {
		result, err := methods.signIn(device, database, username, password)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	err := initUsers()
	if err != nil {
		return nil, err
	}

	item, err := users.
		Selects().
		Where(Eq("username", username)).
		And(Eq("password", password)).
		Rows(nil)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, fmt.Errorf(msg.MSG_AUTHENTICATION_FAILED)
	}

	return newSession(device, username)
}

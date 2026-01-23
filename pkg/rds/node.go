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
* @param database, schema, name, host string
* @return *Model, error
**/
func (s *Node) getModel(database, schema, name, host string) (*Model, error) {
	if !s.started {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(database, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	key := modelKey(database, schema, name)
	s.mu.Lock()
	result, ok := s.models[key]
	s.mu.Unlock()
	if ok {
		return result, nil
	}

	leader, isCluster, err := s.leader()
	if err != nil {
		return nil, err
	}

	if leader != s.host {
		result, err := methods.getModel(leader, database, schema, name, host)
		if err != nil {
			return nil, err
		}

		ok, err = s.loadModel(result)
		if err != nil {
			return nil, err
		}
		if !ok {
			return s.getModel(database, schema, name, host)
		}

		return result, nil
	}

	err = initModels(s.host)
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

	s.mu.Lock()
	s.models[key] = result
	s.mu.Unlock()
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
	ok, err := s.saveModel(model)
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

	s.mu.Lock()
	s.models[model.key()] = model
	s.mu.Unlock()

	return true, nil
}

/**
* saveModel: Saves the model
* @param model *Model
* @return error
**/
func (s *Node) saveModel(model *Model) (bool, error) {
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
		err := methods.saveModel(leader, model)
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

	s.models[key] = model

	return nil
}

/**
* getDB
* @param name string
* @return *DB, error
**/
func (s *Node) getDB(name string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	if s.leader() != s.host {
		result, err := methods.getDB(name)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	result, ok := s.dbs[name]
	if ok {
		return result, nil
	}

	err := initDatabases()
	if err != nil {
		return nil, err
	}

	exists, err := databases.get(name, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf(msg.MSG_DB_NOT_FOUND)
	}

	s.dbs[name] = result
	return result, nil
}

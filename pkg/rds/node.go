package rds

import (
	"encoding/gob"
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

var nodes *Model

/**
* initNodes: Initializes the nodes model
* @param db *DB
* @return error
**/
func initNodes() error {
	if nodes != nil {
		return nil
	}

	db, err := newDb(packageName, node.version)
	if err != nil {
		return err
	}

	nodes, err = db.newModel("", "nodes", true, 1)
	if err != nil {
		return err
	}
	if err := nodes.init(); err != nil {
		return err
	}

	return nil
}

type Node struct {
	host    string             `json:"-"`
	port    int                `json:"-"`
	version string             `json:"-"`
	master  string             `json:"-"`
	rpcs    map[string]et.Json `json:"-"`
	dbs     map[string]*DB     `json:"-"`
	models  map[string]*Model  `json:"-"`
	nodes   map[string]bool    `json:"-"`
	started bool               `json:"-"`
}

/**
* newNode
* @param tp TypeNode, host string, port int, version string
* @return *Node
**/
func newNode(host string, port int, version string) *Node {
	return &Node{
		host:    host,
		port:    port,
		version: version,
		rpcs:    make(map[string]et.Json),
		dbs:     make(map[string]*DB),
		models:  make(map[string]*Model),
		nodes:   make(map[string]bool),
	}
}

/**
* toJson: Converts the node to a json
* @return et.Json
**/
func (s *Node) toJson() et.Json {
	return et.Json{
		"host":    s.host,
		"post":    s.port,
		"version": s.version,
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

		logs.Logf("rpc", "RPC:%s", name)
	}

	rpc.Register(services)
	return nil
}

/**
* addNode
* @param node string
**/
func (s *Node) addNode(host string, port int, version string) error {
	err := initNodes()
	if err != nil {
		return err
	}

	data := s.toJson()
	key := fmt.Sprintf("%s:%d", host, port)
	err = nodes.putData(key, data)
	if err != nil {
		return err
	}

	s.nodes[key] = true
	return nil
}

/**
* removeNode
* @param node string
**/
func (s *Node) removeNode(host string, port int) error {
	err := initNodes()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%d", host, port)
	err = nodes.removeData(key)
	if err != nil {
		return err
	}

	delete(s.nodes, key)
	return nil
}

/**
* getDb
* @param name string
* @return *DB, error
**/
func (s *Node) getDb(name string) (*DB, error) {
	result, ok := s.dbs[name]
	if ok {
		return result, nil
	}

	if s.master != "" {
		result, err := methods.getDB(name)
		if err != nil {
			return nil, err
		}

		s.dbs[name] = result
		return result, nil
	}

	result, err := getDB(name)
	if err != nil {
		return nil, err
	}

	s.dbs[name] = result
	return result, nil
}

/**
* getModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Node) getModel(database, schema, model string) (*Model, error) {
	key := model
	if schema != "" {
		key = fmt.Sprintf("%s.%s", schema, key)
	}
	if database != "" {
		key = fmt.Sprintf("%s.%s", database, key)
	}

	result, ok := s.models[key]
	if ok {
		return result, nil
	}

	if s.master != "" {
		result, err := methods.getModel(database, schema, model)
		if err != nil {
			return nil, err
		}

		s.models[key] = result
		return result, nil
	}

	result, err := getModel(database, schema, model)
	if err != nil {
		return nil, err
	}

	s.models[key] = result
	return result, nil
}

func init() {
	gob.Register(time.Time{})
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(&DB{})
	gob.Register(&Schema{})
	gob.Register(&Model{})
	gob.Register(&Session{})
	gob.Register(&Tx{})
}

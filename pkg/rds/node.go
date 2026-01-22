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

type Node struct {
	host    string          `json:"-"`
	port    int             `json:"-"`
	version string          `json:"-"`
	master  string          `json:"-"`
	dbs     map[string]*DB  `json:"-"`
	nodes   map[string]bool `json:"-"`
	started bool            `json:"-"`
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
		dbs:     make(map[string]*DB),
		nodes:   make(map[string]bool),
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

		logs.Debug(et.Json{
			"method":  metodo,
			"inputs":  inputs,
			"outputs": outputs,
		}.ToString())

	}

	rpc.Register(services)
	return nil
}

/**
* addNode
* @param node string
**/
func (s *Node) addNode(node string) {
	s.nodes[node] = true
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
		return methods.getDB(name)
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
	db, err := s.getDb(database)
	if err != nil {
		return nil, err
	}

	return db.getModel(schema, model)
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

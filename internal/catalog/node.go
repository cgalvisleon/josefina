package catalog

import (
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/tcp"
)

type Node struct {
	*tcp.Server
	app       string              `json:"-"`
	version   string              `json:"-"`
	isStrict  bool                `json:"-"`
	started   bool                `json:"-"`
	dbs       map[string]*DB      `json:"-"`
	models    map[string]*Model   `json:"-"`
	sessions  map[string]*Session `json:"-"`
	muDB      sync.RWMutex        `json:"-"`
	muModel   sync.RWMutex        `json:"-"`
	muSession sync.RWMutex        `json:"-"`
	isDebug   bool                `json:"-"`
}

/**
* newNode
* @param port int
* @return *Node
**/
func newNode(port int) *Node {
	config, err := getConfig()
	if err != nil {
		return nil
	}

	result := &Node{
		Server:    tcp.NewServer(port),
		app:       "josefina",
		version:   "0.0.1",
		isStrict:  config.IsStrict,
		dbs:       make(map[string]*DB),
		models:    make(map[string]*Model),
		sessions:  make(map[string]*Session),
		muDB:      sync.RWMutex{},
		muModel:   sync.RWMutex{},
		muSession: sync.RWMutex{},
	}

	result.Mount(new(Lead))

	return result
}

/**
* toJson: Converts the node to a json
* @return et.Json
**/
func (s *Node) toJson() et.Json {
	leader, imLeader := s.LeaderID()
	return et.Json{
		"app":       s.app,
		"version":   s.version,
		"is_strict": s.isStrict,
		"address":   s.Address(),
		"port":      s.Port(),
		"leader":    leader,
		"im_leader": imLeader,
		"peers":     s.Peers,
	}
}

/**
* Start: Starts the node
* @return error
**/
func (s *Node) Start() error {
	if s.started {
		return nil
	}

	nodes, err := getNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		s.AddNode(node)
	}

	err = s.Start()
	if err != nil {
		return err
	}

	s.started = true

	return nil
}

/**
* LoadModel: Loads a model
* @param model *Model
* @return error
**/
func (s *Node) LoadModel(model *Model) error {
	model.IsInit = false
	err := model.Init()
	if err != nil {
		return err
	}

	return nil
}

/**
* isExisted: Checks if the object exists
* @param from *From, field, idx string
* @return bool, error
**/
func isExisted(from *From, field, idx string) (bool, error) {
	return false, nil
}

/**
* removeObject
* @param from *From, idx string
* @return error
**/
func removeObject(from *From, idx string) error {
	return nil
}

/**
* putObject
* @param from *From, idx string, data et.Json
* @return error
**/
func putObject(from *From, idx string, data et.Json) error {
	return nil
}

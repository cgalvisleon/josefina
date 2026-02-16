package catalog

import (
	"errors"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/msg"
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
	result.Mount(new(Follow))

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
* GetNode: Gets a node
* @param addr string
* @return *tcp.Client, error
**/
func (s *Node) GetNode(addr string) (*tcp.Client, error) {
	idx := slices.IndexFunc(s.Peers, func(item *tcp.Client) bool { return item.Addr == addr })
	if idx == -1 {
		return nil, errors.New(msg.MSG_NODE_NOT_FOUND)
	}

	return s.Peers[idx], nil
}

/**
* isExisted: Checks if the object exists
* @param from *From, field, idx string
* @return bool, error
**/
func isExisted(from *From, field, idx string) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	nd, err := node.GetNode(from.Address)
	if err != nil {
		return false, err
	}

	res := node.Request(nd, "Follow.IsExisted", from, field, idx)
	if res.Error != nil {
		return false, res.Error
	}

	var exists bool
	err = res.Get(&exists)
	if err != nil {
		return false, err
	}

	return false, nil
}

/**
* removeObject
* @param from *From, idx string
* @return error
**/
func removeObject(from *From, idx string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	nd, err := node.GetNode(from.Address)
	if err != nil {
		return err
	}

	res := node.Request(nd, "Follow.RemoveObject", from, idx)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

/**
* putObject
* @param from *From, idx string, data et.Json
* @return error
**/
func putObject(from *From, idx string, data et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	nd, err := node.GetNode(from.Address)
	if err != nil {
		return err
	}

	res := node.Request(nd, "Follow.PutObject", from, idx, data)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

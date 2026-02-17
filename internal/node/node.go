package node

import (
	"errors"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Node struct {
	*tcp.Server
	app       string                    `json:"-"`
	version   string                    `json:"-"`
	isStrict  bool                      `json:"-"`
	started   bool                      `json:"-"`
	dbs       map[string]*catalog.DB    `json:"-"`
	models    map[string]*catalog.Model `json:"-"`
	sessions  map[string]*Session       `json:"-"`
	cache     map[string][]byte         `json:"-"`
	muDB      sync.RWMutex              `json:"-"`
	muModel   sync.RWMutex              `json:"-"`
	muSession sync.RWMutex              `json:"-"`
	muCache   sync.RWMutex              `json:"-"`
	lead      *Lead                     `json:"-"`
	follow    *Follow                   `json:"-"`
	isDebug   bool                      `json:"-"`
}

const (
	appName = "josefina"
	version = "0.0.1"
)

var nodes *Node

/**
* Load: Loads the node
* @param port int
* @return *Node
**/
func Load(port int) *Node {
	if node != nil {
		return node
	}

	config, err := getConfig()
	if err != nil {
		return nil
	}

	result := &Node{
		Server:    tcp.NewServer(port),
		app:       appName,
		version:   version,
		isStrict:  config.IsStrict,
		dbs:       make(map[string]*catalog.DB),
		models:    make(map[string]*catalog.Model),
		sessions:  make(map[string]*Session),
		cache:     make(map[string][]byte),
		muDB:      sync.RWMutex{},
		muModel:   sync.RWMutex{},
		muSession: sync.RWMutex{},
		muCache:   sync.RWMutex{},
	}
	result.lead = &Lead{node: result}
	result.follow = &Follow{node: result}
	result.Mount(result.lead)
	result.Mount(result.follow)

	return result
}

/**
* ToJson: Converts the node to a json
* @return et.Json
**/
func (s *Node) ToJson() et.Json {
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
* IsExisted: Checks if the object exists
* @param from *From, field, idx string
* @return bool, error
**/
func (s *Node) IsExisted(from *catalog.From, field, idx string) (bool, error) {
	nd, err := s.GetNode(from.Address)
	if err != nil {
		return false, err
	}

	res := s.Request(nd, "Follow.IsExisted", from, field, idx)
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
* RemoveObject
* @param from *From, idx string
* @return error
**/
func (s *Node) RemoveObject(from *catalog.From, idx string) error {
	nd, err := s.GetNode(from.Address)
	if err != nil {
		return err
	}

	res := s.Request(nd, "Follow.RemoveObject", from, idx)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

/**
* PutObject
* @param from *From, idx string, data et.Json
* @return error
**/
func (s *Node) PutObject(from *catalog.From, idx string, data et.Json) error {
	nd, err := s.GetNode(from.Address)
	if err != nil {
		return err
	}

	res := s.Request(nd, "Follow.PutObject", from, idx, data)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

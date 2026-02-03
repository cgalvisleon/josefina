package jql

import (
	"sync"

	"github.com/cgalvisleon/josefina/internal/mod"
)

type Node struct {
	address   string
	dbs       map[string]*mod.DB
	models    map[string]*mod.Model
	modelMu   sync.RWMutex
	isStrict  bool
	getLeader func() (string, bool)
	nextHost  func() string
}

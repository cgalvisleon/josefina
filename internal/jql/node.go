package jql

import (
	"sync"

	"github.com/cgalvisleon/josefina/internal/catalog"
)

type Node struct {
	address   string
	dbs       map[string]*catalog.DB
	models    map[string]*catalog.Model
	modelMu   sync.RWMutex
	isStrict  bool
	getLeader func() (string, bool)
	nextHost  func() string
}

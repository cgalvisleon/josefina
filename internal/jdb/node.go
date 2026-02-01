package jdb

import "github.com/cgalvisleon/et/et"

type Node interface {
	GetModel(from *From) (*Model, error)
	IsExisted(from *From, field, idx string) (bool, error)
	RemoveObject(from *From, idx string) error
	PutObject(from *From, idx string, data et.Json) error
}

var node Node

func SetNode(n Node) {
	node = n
}

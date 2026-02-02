package jdb

type Node interface {
	GetModel(from *From) (*Model, error)
	IsExisted(from *From, field, idx string) (bool, error)
}

var node Node

func SetNode(n Node) {
	node = n
}

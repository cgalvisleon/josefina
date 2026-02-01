package core

type Node interface{}

var node Node

func SetNode(n Node) {
	node = n
}

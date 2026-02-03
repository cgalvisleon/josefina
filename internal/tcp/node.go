package tcp

import "sync/atomic"

type Node struct {
	Address string
	Alive   atomic.Bool
	Conns   atomic.Int64
}

func NewNode(addr string) *Node {
	n := &Node{Address: addr}
	n.Alive.Store(true)
	return n
}

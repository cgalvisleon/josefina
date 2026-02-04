package tcp

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Mode int

const (
	ModeServer Mode = iota
	ModeBalancer
)

type Server struct {
	port  int
	nodes []*Node
	b     *Balancer
	mode  atomic.Value
}

func NewServer(port int) *Server {
	result := &Server{
		port:  port,
		nodes: []*Node{},
	}
	result.mode.Store(ModeServer)
	return result
}

/**
* setMode
* @param m Mode
**/
func (s *Server) setMode(m Mode) {
	s.mode.Store(m)
}

/**
* AddNode
* @param address string
**/
func (s *Server) AddNode(address string) {
	node := NewNode(address)
	s.nodes = append(s.nodes, node)
}

/**
* handle
* @param conn net.Conn
**/
func (s *Server) handle(conn net.Conn) {
	mode := s.mode.Load().(Mode)

	switch mode {
	case ModeBalancer:
		s.handleBalancer(conn)
	default:
		s.handleServer(conn)
	}
}

/**
* Listen
* @return error
**/
func (s *Server) Listen() error {
	address := fmt.Sprintf(":%d", s.port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	logs.Logf("TCP", msg.MSG_TCP_LISTENING, s.port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go s.handle(conn)
	}
}

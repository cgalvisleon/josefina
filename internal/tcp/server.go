package tcp

import (
	"fmt"
	"net"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type NodeState int

const (
	Follower NodeState = iota
	Leader
)

type Server struct {
	port  int
	nodes []*Node
	b     *Balancer
	state NodeState
	ln    net.Listener
}

func NewServer(port int) *Server {
	return &Server{
		port:  port,
		nodes: []*Node{},
	}
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
* Start
* @param state NodeState
* @return error
**/
func (s *Server) Start(state NodeState) error {
	var err error
	s.state = state
	if s.ln != nil {
		err = s.ln.Close()
		if err != nil {
			return err
		}
	}

	s.ln, err = net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}

	if s.state == Leader {
		logs.Logf("TCP", msg.MSG_TCP_LISTENING, s.port)
		s.b = NewBalancer(s.nodes)
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				continue
			}

			go Proxy(conn, s.b)
		}
	} else {

	}

	return nil
}

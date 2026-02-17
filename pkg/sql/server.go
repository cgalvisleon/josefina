package sql

import (
	"github.com/cgalvisleon/josefina/internal/jdb"
)

type Server struct {
	node    *jdb.Node
	started bool
}

/**
* NewServer
* @param port int
* @return *Server
**/
func NewServer(port int) *Server {
	result := &Server{
		node: jdb.Load(port),
	}

	return result
}

/**
* Start
* @return error
**/
func (s *Server) Start() error {
	if s.started {
		return nil
	}

	err := s.node.Start()
	if err != nil {
		return err
	}

	s.started = true
	return nil
}

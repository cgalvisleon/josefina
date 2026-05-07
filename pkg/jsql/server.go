package jsql

import (
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/josefina/internal/jdb"
)

type Server struct {
	*jdb.Node
	started bool
}

var srv *Server

/**
* NewServer
* @param port int
* @return *Server, error
**/
func NewServer(port int) (*Server, error) {
	if srv != nil {
		return srv, nil
	}

	n, err := jdb.Load(port)
	if err != nil {
		return nil, err
	}

	srv = &Server{
		Node: n,
	}

	return srv, nil
}

/**
* Start
* @return error
**/
func (s *Server) Start() error {
	if s.started {
		return nil
	}

	err := s.Node.Start()
	if err != nil {
		return err
	}

	s.started = true
	return nil
}

/**
* Authenticate
* @param token string
* @return *claim.Claim, error
**/
func (s *Server) Authenticate(token string) (*claim.Claim, error) {
	return s.Node.Authenticate(token)
}

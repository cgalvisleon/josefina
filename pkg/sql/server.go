package sql

import (
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/josefina/internal/jdb"
)

type Server struct {
	node    *jdb.Node
	started bool
}

var srv *Server

/**
* NewServer
* @param port int
* @return *Server
**/
func NewServer(port int) *Server {
	if srv != nil {
		return srv
	}

	srv = &Server{
		node: jdb.Load(port),
	}

	return srv
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

/**
* Authenticate
* @param token string
* @return *claim.Claim, error
**/
func (s *Server) Authenticate(token string) (*claim.Claim, error) {
	return s.node.Authenticate(token)
}

/**
* SignIn
* @param device, username, password string, tpConn jdb.TpConnection, database string
* @return *Session, error
**/
func (s *Server) SignIn(device, username, password string, tpConn jdb.TpConnection, database string) (*jdb.Session, error) {
	return s.node.SignIn(device, username, password, tpConn, database)
}

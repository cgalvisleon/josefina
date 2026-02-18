package sql

import (
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/tcp"
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
* @return *Server
**/
func NewServer(port int) *Server {
	if srv != nil {
		return srv
	}

	srv = &Server{
		Node: jdb.Load(port),
	}

	srv.OnInbound(func(c *tcp.Client, m *tcp.Message) {
		logs.Debug(et.Json{
			"client":  c,
			"message": m,
		}.ToString())
	})

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

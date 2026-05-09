package server

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/jsql"
)

type Service struct {
	port int
	node *jsql.Server
}

/**
* New
* @return *Service
**/
func New(port int) *Service {
	if port == 0 {
		port = envar.GetInt("PORT", 1377)
	}
	srv, err := jsql.NewServer(port)
	if err != nil {
		logs.Panic(err)
	}
	return &Service{
		port: port,
		node: srv,
	}
}

/**
* Start
**/
func (s *Service) Start() {
	if err := s.node.Start(); err != nil {
		logs.Error(err)
		return
	}
	utility.AppWait()
}

/**
* Stop
**/
func (s *Service) Stop() {
	if err := s.node.Close(); err != nil {
		logs.Error(err)
		return
	}
}

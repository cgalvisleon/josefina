package server

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/jsql"
)

type Service struct {
	node *jsql.Server
}

/**
* New
* @return *Service
**/
func New() *Service {
	port := envar.GetInt("PORT", 1377)
	srv, err := jsql.NewServer(port)
	if err != nil {
		logs.Panic(err)
	}
	return &Service{node: srv}
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

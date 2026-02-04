package http

import (
	"github.com/cgalvisleon/et/server"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/ws"
	v1 "github.com/cgalvisleon/josefina/internal/services/v1"
)

type Service struct {
	ettp *server.Ettp
	ws   *ws.Hub
	tcp  *tcp.Server
}

func New() *Service {
	result := &Service{
		ettp: server.New(v1.PackageName),
	}

	latest := v1.New()
	result.ettp.Mount("/", latest)
	result.ettp.Mount("/v1", latest)
	result.ettp.OnClose(v1.Close)

	return result
}

/**
* Start
* @return
**/
func (s *Service) Start() {
	s.ettp.Start()
}

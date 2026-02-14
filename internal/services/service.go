package http

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/server"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	v1 "github.com/cgalvisleon/josefina/internal/services/v1"
	"github.com/cgalvisleon/josefina/pkg/websocket"
)

var (
	app = "josefina"
)

type Service struct {
	ettp *server.Ettp
	ws   *ws.Hub
}

func New() *Service {
	port := envar.GetInt("HTTP_PORT", 3500)
	result := &Service{
		ettp: server.New(app, port),
		ws:   websocket.New(),
	}

	result.ettp.OnClose(v1.Close)
	latest := v1.Api()
	result.ettp.Mount("/", latest)
	result.ettp.Mount("/v1", latest)

	wsHandler := websocket.Init()
	result.ettp.Mount("/ws", wsHandler)

	return result
}

/**
* Start
* @return
**/
func (s *Service) Start() {
	if s.ettp != nil {
		s.ettp.Start()
	}

	if s.ws != nil {
		s.ws.Start()
	}

	utility.AppWait()

	s.onClose()
}

/**
* onClose
* @return
**/
func (s *Service) onClose() {
	if s.ettp != nil {
		s.ettp.Close()
	}

	if s.ws != nil {
		s.ws.Close()
	}
}

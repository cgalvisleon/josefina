package http

import (
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/server"
	"github.com/cgalvisleon/et/tcp"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	v1 "github.com/cgalvisleon/josefina/internal/services/v1"
	"github.com/cgalvisleon/josefina/pkg/jdb"
	"github.com/cgalvisleon/josefina/pkg/websocket"
)

var (
	appName = "josefina"
)

type Service struct {
	ettp *server.Ettp
	ws   *ws.Hub
	tcp  *tcp.Server
}

func New() *Service {
	result := &Service{
		ettp: server.New(appName),
		ws:   websocket.New(),
	}

	err := jdb.Load()
	if err != nil {
		logs.Panic(err)
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

func (s *Service) onClose() {
	if s.ettp != nil {
		s.ettp.Close()
	}
	if s.ws != nil {
		s.ws.Close()
	}
}

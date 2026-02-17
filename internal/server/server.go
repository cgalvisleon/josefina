package server

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/server"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"

	api "github.com/cgalvisleon/josefina/pkg/http"
	"github.com/cgalvisleon/josefina/pkg/jdb"
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
	err := jdb.Load()
	if err != nil {
		logs.Panic(err)
	}

	port := envar.GetInt("HTTP_PORT", 3500)
	result := &Service{
		ettp: server.New(app, port),
		ws:   websocket.New(),
	}

	latest := api.Init()
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

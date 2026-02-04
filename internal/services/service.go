package http

import (
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/server"
	"github.com/cgalvisleon/et/tcp"
	v1 "github.com/cgalvisleon/josefina/internal/services/v1"
	"github.com/cgalvisleon/josefina/pkg/jdb"
	ws "github.com/cgalvisleon/josefina/pkg/ws"
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
	}

	err := jdb.Load()
	if err != nil {
		logs.Panic(err)
	}

	result.ettp.OnClose(v1.Close)

	latest := v1.Api()
	result.ettp.Mount("/", latest)
	result.ettp.Mount("/v1", latest)

	wsHandler := v1.Ws()
	result.ettp.Mount("/ws", wsHandler)

	return result
}

/**
* Start
* @return
**/
func (s *Service) Start() {
	s.ettp.Start()
}

/**
* Ws
* @return http.Handler
**/
func (s *Service) websocket() http.Handler {
	result := ws.Init(s.ws)
	return result
}

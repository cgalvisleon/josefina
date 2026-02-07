package websocket

import (
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/ws"
	"github.com/cgalvisleon/josefina/pkg/jdb"
	"github.com/go-chi/chi/v5"
)

var (
	hub *ws.Hub
)

func New() *ws.Hub {
	if hub == nil {
		hub = ws.New()
	}

	hub.OnListener(onListener)

	return hub
}

func Init() http.Handler {
	hub.OnConnection(func(sub *ws.Client) {
		logs.Debug("Connection:", sub.Name)
	})
	hub.OnDisconnection(func(sub *ws.Client) {
		logs.Debug("Disconnection:", sub.Name)
	})

	r := chi.NewRouter()
	r.Get("/", WsUpgrader)
	r.With(jdb.Authenticate).Post("/topic", HttpTopic)
	r.With(jdb.Authenticate).Post("/queue", HttpQueue)
	r.With(jdb.Authenticate).Post("/stack", HttpStack)
	r.With(jdb.Authenticate).Post("/remove", HttpRemove)
	r.With(jdb.Authenticate).Post("/subscribe", HttpSubscribe)
	return r
}

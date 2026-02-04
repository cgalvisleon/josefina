package websocket

import (
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/ws"
	"github.com/go-chi/chi/v5"
)

var (
	hub *ws.Hub
)

func New() *ws.Hub {
	if hub == nil {
		hub = ws.New()
	}

	return hub
}

func Init() http.Handler {
	hub.OnConnection(func(sub *ws.Subscriber) {
		logs.Debug("Connection:", sub.Name)
	})
	hub.OnDisconnection(func(sub *ws.Subscriber) {
		logs.Debug("Disconnection:", sub.Name)
	})

	r := chi.NewRouter()
	r.Get("/", WsUpgrader)
	r.Post("/topic", HttpTopic)
	r.Post("/queue", HttpQueue)
	r.Post("/stack", HttpStack)
	r.Post("/remove", HttpRemove)
	r.Post("/subscribe", HttpSubscribe)
	return r
}

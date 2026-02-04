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

func Init(h *ws.Hub) http.Handler {
	h.OnConnection(func(sub *ws.Subscriber) {
		logs.Debug("Connection:", sub.Name)
	})
	h.OnDisconnection(func(sub *ws.Subscriber) {
		logs.Debug("Disconnection:", sub.Name)
	})

	hub = h
	r := chi.NewRouter()
	r.Get("/", WsUpgrader)
	r.Post("/topic", HttpTopic)
	r.Post("/queue", HttpQueue)
	r.Post("/stack", HttpStack)
	r.Post("/remove", HttpRemove)
	r.Post("/subscribe", HttpSubscribe)
	return r
}

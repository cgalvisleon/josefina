package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Init() http.Handler {
	r := chi.NewRouter()
	r.Get("/", WsUpgrader)
	r.Post("/topic", HttpTopic)
	r.Post("/queue", HttpQueue)
	r.Post("/stack", HttpStack)
	r.Post("/remove", HttpRemove)
	r.Post("/subscribe", HttpSubscribe)
	return r
}

package v1

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/josefina/pkg/jdb"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	ctx         context.Context
	PackageName string
	PackagePath string
	Hostname    string
}

func newRouter(name string) *Router {
	hostname, _ := os.Hostname()
	return &Router{
		ctx:         context.Background(),
		PackageName: name,
		PackagePath: envar.GetStr("PATH_URL", "/api/josefina"),
		Hostname:    hostname,
	}
}

/**
* Routes
* @return http.Handler
**/
func (s *Router) Routes() http.Handler {
	host := strs.Format("http://%s", envar.GetStr("HOST", "localhost"))
	host = strs.Format("%s:%d", host, envar.GetInt("PORT", 3300))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	router.SetAutentication(jdb.Authenticate)
	router.Public(r, router.Get, "/version", s.version, s.PackageName, s.PackagePath, host)
	router.Public(r, router.Post, "/signin", s.signIn, s.PackageName, s.PackagePath, host)
	router.Private(r, router.Post, "/jql", s.jql, s.PackageName, s.PackagePath, host)

	middleware.SetServiceName(s.PackageName)
	return r
}

/**
* WsRouter
* @return http.Handler
**/
func (s *Router) WsRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/ws", jdb.WsUpgrader)
	r.Post("/ws/topic", jdb.HttpTopic)
	r.Post("/ws/queue", jdb.HttpQueue)
	r.Post("/ws/stack", jdb.HttpStack)
	r.Post("/ws/remove", jdb.HttpRemove)
	r.Post("/ws/subscribe", jdb.HttpSubscribe)
	return r
}

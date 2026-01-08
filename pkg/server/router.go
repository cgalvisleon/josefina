package server

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

const (
	Version     = "1.0.0"
	PackageName = "josefina"
)

var (
	PackagePath = envar.GetStr("PATH_API", "/josefina")
	Hostname, _ = os.Hostname()
)

type Router struct {
	ctx context.Context
}

func NewRouter() *Router {
	return &Router{
		ctx: context.Background(),
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
	router.Public(r, router.Get, "/version", s.version, PackageName, PackagePath, host)

	middleware.SetServiceName(PackageName)
	logs.Logf(PackageName, "Router version:%s", config.App.Version)

	return r
}

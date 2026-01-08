package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

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

type Router struct {
	ctx         context.Context
	PackagePath string
	Hostname    string
}

func NewRouter() *Router {
	hostname, _ := os.Hostname()
	return &Router{
		ctx:         context.Background(),
		PackagePath: envar.GetStr("PATH_API", "/api/josefina"),
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
	router.Public(r, router.Get, "/version", s.version, PackageName, s.PackagePath, host)

	middleware.SetServiceName(PackageName)
	path := fmt.Sprintf("%s%s", host, s.PackagePath)
	logs.Logf(PackageName, "Router version:%s url:%s host:%s", Version, path, s.Hostname)

	return r
}

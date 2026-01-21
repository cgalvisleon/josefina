package v1

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	ctx         context.Context
	Version     string
	PackageName string
	PackagePath string
	Hostname    string
}

func newRouter(name, version string) *Router {
	hostname, _ := os.Hostname()
	return &Router{
		ctx:         context.Background(),
		Version:     version,
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
	router.Public(r, router.Get, "/version", s.version, s.PackageName, s.PackagePath, host)

	middleware.SetServiceName(s.PackageName)
	return r
}

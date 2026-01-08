package server

import (
	"context"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/go-chi/chi/v5"
)

const (
	Version     = "1.0.0"
	PackageName = "apps"
)

var (
	PackagePath = envar.GetStr("PATH_API", "/josefina")
	Hostname, _ = os.Hostname()
)

type Router struct {
	Ctx context.Context
}

/**
* version
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) version(w http.ResponseWriter, r *http.Request) {
	result, err := version(s.Ctx)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, result)
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

package server

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
)

type Router struct {
	*router.Api
	Db *jsql.DB
}

var api *Router

func Routes(name string, version string, db *jsql.DB) http.Handler {
	if api != nil {
		return nil
	}

	host := envar.GetStr("HOST", "localhost")
	port := envar.GetInt("PORT", 2000)
	pathUrl := envar.GetStr("PATH_URL", "/josephine")
	rpc := envar.GetInt("RPC", 4200)
	r := router.NewApi(name, pathUrl, host, port, rpc, version)
	r.UseAuthentication(middleware.Authentication)

	api = &Router{
		Api: r,
		Db:  db,
	}
	api.Public(router.GET, "/version", api.version)

	api.init()

	logs.Logf(api.Name, "Router version:%s", api.Version)

	return api.Router
}

/**
* init
**/
func (s *Router) init() {
	s.initEvents()
}

/**
* version
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) version(w http.ResponseWriter, r *http.Request) {
	result := et.Json{
		"version": s.Version,
		"service": s.Name,
		"host":    s.Host,
		"company": "Qdra",
		"web":     envar.GetStr("WEB", "https://company.com"),
		"help":    envar.GetStr("HELP", "https://company.com/help"),
	}

	response.JSON(w, r, http.StatusOK, result)
}

/**
* routes
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) routes(w http.ResponseWriter, r *http.Request) {
	_routes := router.GetRoutes()
	routes := []et.Json{}
	for _, route := range _routes {
		routes = append(routes, et.Json{
			"method": route.Str("method"),
			"path":   route.Str("path"),
		})
	}

	result := et.Items{
		Ok:     true,
		Count:  len(routes),
		Result: routes,
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

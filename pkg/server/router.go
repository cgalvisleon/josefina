package server

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
	"github.com/josefina/internal/jdb"
)

type Router struct {
	*router.Api
	Server *jdb.Server
}

var api *Router

func Routes(name string, version string, srv *jdb.Server) http.Handler {
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
		Api:    r,
		Server: srv,
	}
	api.Public(router.GET, "/version", api.version)
	api.Authentication(router.GET, "/routes", api.routes)
	// JDB
	api.Public(router.GET, "/signin", api.jdbSignin)

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
	rutas := router.GetRoutes()
	routes := []et.Json{}
	for _, route := range rutas {
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

/**
* jdbSignin
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) jdbSignin(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	database := body.Str("database")
	username := body.Str("username")
	password := body.Str("password")
	item, err := signin(database, username, password)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, item)
}

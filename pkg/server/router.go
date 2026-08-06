package server

import (
	"net/http"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
	"github.com/josefina/internal/jdb"
	"github.com/josefina/internal/msg"
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
	rpc := envar.GetInt("RPC_PORT", 4200)
	r := router.NewApi(name, pathUrl, host, port, rpc, version)
	r.UseAuthentication(Authentication)

	api = &Router{
		Api:    r,
		Server: srv,
	}
	api.Public(router.GET, "/version", api.version)
	api.Authentication(router.GET, "/routes", api.routes)
	// JDB
	api.Public(router.POST, "/signin", api.signin)
	api.Public(router.POST, "/signout", api.signout)
	api.Public(router.POST, "/system", api.system)
	api.Public(router.POST, "/query", api.query)
	api.Public(router.POST, "/command", api.command)
	api.Public(router.POST, "/uploadXls", api.uploadXls)

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
		"company": s.Api.Name,
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
* signin
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) signin(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	database := body.Str("database")
	username := body.Str("username")
	password := body.Str("password")
	item, err := jdb.SignIn(database, username, password)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, item)
}

/**
* jdbSignout
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) signout(w http.ResponseWriter, r *http.Request) {
	token, err := GetBearerToken(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	result, err := jdb.SignOut(token)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	first, err := result.First()
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, first)
}

/**
* system
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) system(w http.ResponseWriter, r *http.Request) {
	token, err := GetBearerToken(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	body.Set("token", token)
	result, err := jdb.System(body)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

/**
* query
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) query(w http.ResponseWriter, r *http.Request) {
	token, err := GetBearerToken(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	body.Set("token", token)
	result, err := jdb.JQuery(body)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

/**
* command
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) command(w http.ResponseWriter, r *http.Request) {
	token, err := GetBearerToken(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	body.Set("token", token)
	result, err := jdb.JCommand(body)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

/**
* uploadXls
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Router) uploadXls(w http.ResponseWriter, r *http.Request) {
	token, err := GetBearerToken(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	msxLimitSize := envar.GetInt64("MSX_LIMIT_SIZE", 128<<20)
	err = r.ParseMultipartForm(msxLimitSize) // Limitar el tamaño a 64 MB
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, "Unable to parse form")
		return
	}

	fileData, _, err := r.FormFile("file")
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, msg.MSG_FILE_NOT_FOUND)
		return
	}
	defer fileData.Close()

	params := et.Json{
		"token":   token,
		"file":    fileData,
		"sheet":   r.FormValue("sheet"),
		"idField": r.FormValue("idField"),
		"schema":  r.FormValue("schema"),
		"model":   r.FormValue("model"),
		"atribs":  r.FormValue("atribs"),
	}

	result, err := jdb.JUploadXls(params)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

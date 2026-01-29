package v1

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/middleware"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/router"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/jdb"
	"github.com/cgalvisleon/josefina/pkg/msg"
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
* autentication
* @param next http.Handler
* @return http.Handler
**/
func (s *Router) autentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if !utility.ValidStr(token, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
			return
		}

		token = utility.PrefixRemove("Bearer", token)
		result, err := claim.ParceToken(token)
		if err != nil {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
			return
		}

		key := fmt.Sprintf("%s:%s:%s", result.App, result.Device, result.Username)
		session, exists := jdb.GetCacheStr(key)
		if !exists {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
			return
		}

		if session != token {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
			return
		}

		ctx := r.Context()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
	router.SetAutentication(s.autentication)
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
	return r
}

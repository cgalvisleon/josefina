package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var (
	PackageName = "josefina"
)

/**
* Init
* @return http.Handler
**/
func Init() http.Handler {
	r := chi.NewRouter()
	rt := newRouter(PackageName)
	r.Mount(rt.PackagePath, rt.Routes())

	return r
}

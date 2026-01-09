package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	pkg "github.com/cgalvisleon/josefina/pkg/server/v1"
	"github.com/go-chi/chi/v5"
)

var (
	PackageName = "josefina"
	Version     = "1.0.0"
)

/**
* New
* @return http.Handler
**/
func New() http.Handler {
	r := chi.NewRouter()

	err := pkg.InitJosefina()
	if err != nil {
		logs.Log(PackageName, err)
	}

	server := pkg.NewRouter(PackageName, Version)
	r.Mount(server.PackagePath, server.Routes())

	return r
}

/**
* Close
**/
func Close() {
	jrpc.Close()
}

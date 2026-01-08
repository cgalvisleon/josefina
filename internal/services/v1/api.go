package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	pkg "github.com/cgalvisleon/josefina/pkg/server"
	"github.com/go-chi/chi"
)

/**
* New
* @return http.Handler
**/
func New() http.Handler {
	r := chi.NewRouter()

	err := pkg.InitCore()
	if err != nil {
		logs.Log(pkg.PackageName, err)
	}

	server := pkg.NewRouter()
	r.Mount(server.RootPath, server.Routes())

	return r
}

/**
* Close
**/
func Close() {
	jrpc.Close()
}

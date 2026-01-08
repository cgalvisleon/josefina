package v1

import (
	"context"
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/event"
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

	server := &pkg.Router{
		Ctx: context.Background(),
	}
	r.Mount(config.App.PathApi, server.Routes())

	return r
}

/**
* Close
**/
func Close() {
	jrpc.Close()
	cache.Close()
	event.Close()
}

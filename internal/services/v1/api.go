package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/go-chi/chi"
)

func New() http.Handler {
	r := chi.NewRouter()

	err := initCore()
	if err != nil {
		logs.Log(PackageName, err)
	}

	_pkg := &Router{
		Repository: &Controller{},
	}

	r.Mount(config.App.PathApi, _pkg.Routes())

	return r
}

func Close() {
	jrpc.Close()
	cache.Close()
	event.Close()
}

package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/pkg/server"
)

var (
	AppName = "josefina"
)

func New() http.Handler {
	err := cache.Load()
	if err != nil {
		logs.Panic(err)
	}

	err = event.Load()
	if err != nil {
		logs.Panic(err)
	}

	db, err := jsql.LoadTo("josephine")
	if err != nil {
		logs.Panic(err)
	}

	api := server.Routes(AppName, "v1.0.0", db)
	return api
}

func Close() {
	jrpc.Close()
	cache.Close()
	event.Close()
}

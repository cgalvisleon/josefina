package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/jdb"
	srv "github.com/josefina/pkg/server"
)

var (
	AppName = "josefina"
)

func New() http.Handler {
	server, err := jdb.Load()
	if err != nil {
		logs.Panic(err)
	}

	api := srv.Routes(AppName, server.Version, server)
	return api
}

func Close() {
	jrpc.Close()
}

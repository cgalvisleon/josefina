package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/josefina/internal/jdb"
	"github.com/josefina/pkg/server"
)

var (
	AppName = "josefina"
)

func New() http.Handler {
	db, err := jdb.Load()
	if err != nil {
		logs.Panic(err)
	}

	api := server.Routes(AppName, db.Version, db)
	return api
}

func Close() {
	jrpc.Close()
}

package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	api "github.com/cgalvisleon/josefina/pkg/http"
	"github.com/cgalvisleon/josefina/pkg/jdb"
)

/**
* Api
* @return http.Handler
**/
func Api() http.Handler {
	err := jdb.Load()
	if err != nil {
		logs.Panic(err)
	}

	result := api.Init()
	return result
}

/**
* Close
**/
func Close() {
	jrpc.Close()
}

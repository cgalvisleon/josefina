package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	api "github.com/cgalvisleon/josefina/pkg/http"
)

/**
* Api
* @return http.Handler
**/
func Api() http.Handler {
	result := api.Init()
	return result
}

/**
* Close
**/
func Close() {
	jrpc.Close()
}

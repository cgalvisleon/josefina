package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	api "github.com/cgalvisleon/josefina/pkg/http"
	ws "github.com/cgalvisleon/josefina/pkg/ws"
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
* Ws
* @return http.Handler
**/
func Ws() http.Handler {
	result := ws.Init()
	return result
}

/**
* Close
**/
func Close() {
	jrpc.Close()
}

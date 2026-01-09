package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	pkg "github.com/cgalvisleon/josefina/pkg/server/v1"
)

var PackageName = pkg.PackageName

/**
* New
* @return http.Handler
**/
func New() http.Handler {
	result, err := pkg.InitJosefina()
	if err != nil {
		logs.Log(pkg.PackageName, err)
	}

	return result
}

/**
* Close
**/
func Close() {
	jrpc.Close()
}

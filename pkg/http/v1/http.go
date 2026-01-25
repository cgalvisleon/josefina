package v1

import (
	"net/http"

	"github.com/cgalvisleon/josefina/pkg/jdb"
	"github.com/go-chi/chi/v5"
)

var (
	PackageName = "josefina"
)

/**
* Init
* @return error
**/
func Init() (http.Handler, error) {
	err := jdb.Load()
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	server := newRouter(PackageName)
	r.Mount(server.PackagePath, server.Routes())

	return r, nil
}

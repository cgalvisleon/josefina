package v1

import (
	"net/http"

	"github.com/cgalvisleon/josefina/pkg/rds"
	"github.com/go-chi/chi/v5"
)

var (
	PackageName = "josefina"
	Version     = "1.0.0"
)

/**
* Init
* @return error
**/
func Init() (http.Handler, error) {
	err := rds.Master(Version)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	server := newRouter(PackageName, Version)
	r.Mount(server.PackagePath, server.Routes())

	return r, nil
}

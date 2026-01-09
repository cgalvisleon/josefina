package v1

import (
	"net/http"

	"github.com/cgalvisleon/josefina/pkg/josefina"
	"github.com/go-chi/chi/v5"
)

var (
	PackageName = "josefina"
	Version     = "1.0.0"
)

/**
* InitJosefina
* @return error
**/
func InitJosefina() (http.Handler, error) {
	err := josefina.Init()
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	server := newRouter(PackageName, Version)
	r.Mount(server.PackagePath, server.Routes())

	return r, nil
}

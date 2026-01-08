package server

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
)

/**
* version
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) version(w http.ResponseWriter, r *http.Request) {
	version := et.Json{
		"service": PackageName,
		"version": Version,
		"host":    Hostname,
		"company": "",
		"web":     "",
	}

	response.JSON(w, r, http.StatusOK, version)
}

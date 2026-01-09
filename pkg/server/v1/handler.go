package v1

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
		"service": s.PackageName,
		"version": s.Version,
		"host":    s.Hostname,
		"company": "",
		"web":     "",
	}

	response.JSON(w, r, http.StatusOK, version)
}

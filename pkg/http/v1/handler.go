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

/**
* signIn
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) signIn(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	username := body.Str("username")
	password := body.Str("password")

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok: true,
		Result: et.Json{
			"username": username,
			"password": password,
		},
	})
}

package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/josefina/internal/jql"
	"github.com/cgalvisleon/josefina/pkg/jdb"
)

/**
* version
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) version(w http.ResponseWriter, r *http.Request) {
	version := jdb.HelpCheck()
	response.ITEM(w, r, http.StatusOK, version)
}

/**
* auth
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) auth(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	device := body.Str("device")
	database := body.Str("database")
	username := body.Str("username")
	password := body.Str("password")
	session, err := jdb.Auth(device, database, username, password)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: session.ToJson(),
	})
}

/**
* jql
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) jQuery(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	result, err := jql.Jquery(ctx, body)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

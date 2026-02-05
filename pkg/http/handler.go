package v1

import (
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
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
* signin
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) signin(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	device := body.Str("device")
	username := body.Str("username")
	password := body.Str("password")
	session, err := jdb.SignIn(device, username, password)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := session.ToJson()
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEM(w, r, http.StatusOK, et.Item{
		Ok:     true,
		Result: result,
	})
}

/**
* Query
* @param w http.ResponseWriter, r *http.Request
* @return error
**/
func (s *Router) query(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetStr(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	result, err := jdb.Query(ctx, body)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

/**
* jQuery
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
	result, err := jdb.JQuery(ctx, body)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	response.ITEMS(w, r, http.StatusOK, result)
}

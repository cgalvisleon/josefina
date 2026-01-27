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

	device := body.Str("device")
	database := body.Str("database")
	username := body.Str("username")
	password := body.Str("password")
	session, err := jdb.SignIn(device, database, username, password)
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
func (s *Router) jql(w http.ResponseWriter, r *http.Request) {
	body, err := response.GetBody(r)
	if err != nil {
		response.HTTPError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	token := r.Header.Get("Authorization")
	query := &jdb.Request{}
	query.SetToken(token)
	query.SetBody(body)
	var res jdb.Response
	jdb.Jql(query, &res)
	if res.Error != nil {
		response.HTTPError(w, r, http.StatusBadRequest, res.Error.Message)
		return
	}

	response.ITEMS(w, r, http.StatusOK, et.Items{
		Ok:     true,
		Result: res.Result,
		Count:  len(res.Result),
	})
}

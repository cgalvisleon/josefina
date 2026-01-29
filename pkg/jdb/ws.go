package jdb

import (
	"net/http"

	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/cgalvisleon/josefina/pkg/ws"
)

/**
* WsUpgrader
* @param w http.ResponseWriter, r *http.Request
**/
func WsUpgrader(w http.ResponseWriter, r *http.Request) {
	if !node.started {
		response.HTTPError(w, r, http.StatusBadRequest, msg.MSG_JOSEFINA_NOT_STARTED)
		return
	}

	conn, err := ws.Upgrader(w, r)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if conn == nil {
		response.HTTPError(w, r, http.StatusInternalServerError, "Connection is nil")
		return
	}

	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, err := node.ws.Connect(conn, ctx)
		if err != nil {
			response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}))
	handler.ServeHTTP(w, r)
}

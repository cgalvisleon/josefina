package jdb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
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

	token := r.Header.Get("Authorization")
	if !utility.ValidStr(token, 0, []string{""}) {
		response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
		return
	}

	token = prefixRemove("Bearer", token)
	result, err := claim.ParceToken(token)
	if err != nil {
		response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
		return
	}

	key := fmt.Sprintf("%s:%s:%s", result.App, result.Device, result.Username)
	session, exists := GetCacheStr(key)
	if !exists {
		response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
		return
	}

	if session != token {
		response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, "app", result.App)
	ctx = context.WithValue(ctx, "device", result.Device)
	ctx = context.WithValue(ctx, "username", result.Username)
	_, err = node.ws.Connect(conn, ctx)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
}

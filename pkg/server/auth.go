package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/jdb"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* Authenticate
* @param next http.Handler
* @return http.Handler
**/
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !node.started {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.MSG_JOSEFINA_NOT_STARTED)
			return
		}

		token := r.Header.Get("Authorization")
		result, err := core.Authenticate(token)
		if err != nil {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.MSG_CLIENT_NOT_AUTHENTICATION)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "sessionId", result.Id)
		ctx = context.WithValue(ctx, "app", result.App)
		ctx = context.WithValue(ctx, "device", result.Device)
		ctx = context.WithValue(ctx, "username", result.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

/**
* SignIn: Authenticate a user
* @param device, username, password string
* @return *Session, error
**/
func SignIn(device, username, password string) (*jdb.Session, error) {
	if !node.started {
		return nil, errors.New(msg.MSG_JOSEFINA_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, errors.New(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, errors.New(msg.MSG_PASSWORD_REQUIRED)
	}

	return core.SignIn(device, username, password)
}

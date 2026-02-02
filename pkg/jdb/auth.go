package jdb

import (
	"context"
	"errors"
	"net/http"

	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* Authenticate
* @param next http.Handler
* @return http.Handler
**/
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		result, err := core.Authenticate(token)
		if err != nil {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Message)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "app", result.App)
		ctx = context.WithValue(ctx, "device", result.Device)
		ctx = context.WithValue(ctx, "username", result.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

/**
* applyMiddleware
* @param middlewares []func(http.Handler) http.Handler, next http.Handler
* @return http.Handler
**/
func applyMiddlewares(handler http.Handler, middlewares []func(http.Handler) http.Handler) http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}

	return handler
}

/**
* applyAuthenticate
* @param handler http.Handler
* @return http.Handler
**/
func applyAuthenticate(handler http.Handler) http.Handler {
	middlewares := []func(http.Handler) http.Handler{
		Authenticate,
	}
	return applyMiddlewares(handler, middlewares)
}

/**
* SignIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func Auth(device, database, username, password string) (*Session, error) {
	if !node.started {
		return nil, errors.New(msg.MSG_JOSEFINA_NOT_STARTED)
	}
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, errors.New(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, errors.New(msg.MSG_PASSWORD_REQUIRED)
	}

	return node.auth(device, database, username, password)
}

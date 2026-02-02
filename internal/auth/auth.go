package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/cache"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* authenticate: Authenticates a user
* @param token string
* @return *claim.Token, error
**/
func authenticate(token string) (*claim.Claim, error) {
	if !utility.ValidStr(token, 0, []string{""}) {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	token = utility.PrefixRemove("Bearer", token)
	result, err := claim.ParceToken(token)
	if err != nil {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	key := fmt.Sprintf("%s:%s:%s", result.App, result.Device, result.Username)
	session, exists := cache.GetStr(key)
	if !exists {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	if session != token {
		return nil, msg.ERROR_CLIENT_NOT_AUTHENTICATION.Error()
	}

	return result, nil
}

/**
* Authenticate
* @param next http.Handler
* @return http.Handler
**/
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		result, err := authenticate(token)
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
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, errors.New(msg.MSG_USERNAME_REQUIRED)
	}
	if !utility.ValidStr(password, 0, []string{""}) {
		return nil, errors.New(msg.MSG_PASSWORD_REQUIRED)
	}

	item, err := core.GetUser(username, password)
	if err != nil {
		return nil, err
	}
	if len(item) == 0 {
		return nil, errors.New(msg.MSG_AUTHENTICATION_FAILED)
	}

	result, err := CreateSession(device, username)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s:%s:%s", appName, device, username)
	_, err = cache.Set(key, result.Token, 0)
	if err != nil {
		return nil, err
	}

	return result, nil
}

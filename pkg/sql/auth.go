package sql

import (
	"context"
	"net/http"

	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* Authenticate
* @param next http.Handler
* @return http.Handler
**/
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !srv.started {
			response.HTTPError(w, r, http.StatusUnauthorized, msg.MSG_JOSEFINA_NOT_STARTED)
			return
		}

		token := r.Header.Get("Authorization")
		result, err := srv.Authenticate(token)
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

package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/request"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/jdb"
)

const (
	SessionKey request.ContextKey = "session"
)

/**
* GetBearerToken
* @param r *http.Request
* @return string, error
**/
func GetBearerToken(r *http.Request) (string, error) {
	_, ok := r.Header["Authorization"]
	if !ok {
		return "", logs.Alertm("Autorization is required")
	}

	token := r.Header.Get("Authorization")
	if token == "" {
		return "", logs.Alertm("Autorization is required")
	}

	if !strings.HasPrefix(token, "Bearer ") {
		return "", logs.Alertm("Autorization is required")
	}

	token = strings.TrimPrefix(token, "Bearer ")
	return token, nil
}

/**
* Authentication
* @param next http.Handler
**/
func Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := GetBearerToken(r)
		if err != nil {
			response.Unauthorized(w, r)
			return
		}

		clm, err := claim.ParceToken(token)
		if err != nil {
			response.Unauthorized(w, r)
			return
		}

		if clm == nil {
			response.Unauthorized(w, r)
			return
		}

		session, err := jdb.Authenticate(token)
		if err != nil {
			response.Unauthorized(w, r)
			return
		}

		serviceId := r.Header.Get("ServiceId")
		if serviceId == "" {
			serviceId = utility.UUID()
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, request.ServiceIdKey, serviceId)
		ctx = context.WithValue(ctx, request.DurationKey, clm.Duration)
		ctx = context.WithValue(ctx, request.DeviceKey, clm.Device)
		ctx = context.WithValue(ctx, request.AppKey, clm.App)
		ctx = context.WithValue(ctx, request.SessionIDKey, clm.SessionID)
		ctx = context.WithValue(ctx, request.NameKey, clm.Name)
		ctx = context.WithValue(ctx, request.PayloadKey, clm.Payload)
		ctx = context.WithValue(ctx, request.TokenKey, token)
		ctx = context.WithValue(ctx, SessionKey, session)
		now := utility.Now()
		data, err := clm.ToJson()
		if err != nil {
			response.Unauthorized(w, r)
			return
		}
		data.Set("service_id", serviceId)
		data.Set("host_name", r.Host)
		data.Set("date_at", now)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

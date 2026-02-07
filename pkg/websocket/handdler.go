package websocket

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/pkg/jdb"
)

/**
* WsUpgrader
* @param w http.ResponseWriter, r *http.Request
**/
func WsUpgrader(w http.ResponseWriter, r *http.Request) {
	socket, err := ws.Upgrader(w, r)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if socket == nil {
		response.HTTPError(w, r, http.StatusInternalServerError, "Connection is nil")
		return
	}

	token := r.Header.Get("Authorization")
	result, err := core.Authenticate(token)
	if err != nil {
		ws.SendError(socket, err)
		socket.Close()
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, "sessionId", result.ID)
	ctx = context.WithValue(ctx, "app", result.App)
	ctx = context.WithValue(ctx, "device", result.Device)
	ctx = context.WithValue(ctx, "username", result.Username)
	_, err = hub.Connect(socket, ctx)
	if err != nil {
		ws.SendError(socket, err)
		socket.Close()
	}
}

/**
* onListener
* @param sub *ws.Subscriber, message []byte
**/
func onListener(sub *ws.Subscriber, message []byte) {
	logs.Debug(sub.ToJson().ToString(), " message:", string(message))
}

/**
* HttpTopic create a topic channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpTopic(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		channel := body.Str("channel")
		if !utility.ValidStr(channel, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "channel").Error())
			return
		}

		hub.Topic(channel)
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpQueue create a queue channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpQueue(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		channel := body.Str("channel")
		if !utility.ValidStr(channel, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "channel").Error())
			return
		}

		hub.Queue(channel)
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpStack create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpStack(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		channel := body.Str("channel")
		if !utility.ValidStr(channel, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "channel").Error())
			return
		}

		hub.Stack(channel)
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpRemove create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpRemove(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		channel := body.Str("channel")
		if !utility.ValidStr(channel, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "channel").Error())
			return
		}

		err = hub.Remove(channel)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpSubscribe create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpSubscribe(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		channel := body.Str("channel")
		if !utility.ValidStr(channel, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "channel").Error())
			return
		}

		ctx := r.Context()
		username := ctx.Value("username").(string)
		err = hub.Subscribe(channel, username)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpUnsubscribe create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpUnsubscribe(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		channel := body.Str("channel")
		if !utility.ValidStr(channel, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "channel").Error())
			return
		}

		ctx := r.Context()
		username := ctx.Value("username").(string)
		err = hub.Unsubscribe(channel, username)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpSendTo create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpSendTo(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		to := body.ArrayStr("to")
		if len(to) == 0 {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "to").Error())
			return
		}

		ctx := r.Context()
		username := ctx.Value("username").(string)
		ms := ws.NewMessage(et.Json{
			"username": username,
		}, to)

		_, err = hub.SendTo(to, ms)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpPublish create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpPublish(w http.ResponseWriter, r *http.Request) {
	handler := jdb.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := response.GetBody(r)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		channel := body.Str("channel")
		if !utility.ValidStr(channel, 0, []string{""}) {
			response.HTTPError(w, r, http.StatusBadRequest, fmt.Errorf(msg.MSG_ARG_REQUIRED, "channel").Error())
			return
		}

		ctx := r.Context()
		username := ctx.Value("username").(string)
		ms := ws.NewMessage(et.Json{
			"username": username,
		}, []string{})
		ms.Channel = channel
		ms.Message = body.Str("message")
		_, err = hub.Publish(channel, ms)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}))
	handler.ServeHTTP(w, r)
}

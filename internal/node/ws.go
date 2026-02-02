package node

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/et/ws"
	"github.com/cgalvisleon/josefina/pkg/msg"
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

/**
* HttpTopic create a topic channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpTopic(w http.ResponseWriter, r *http.Request) {
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		node.ws.Topic(channel)
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpQueue create a queue channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpQueue(w http.ResponseWriter, r *http.Request) {
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		node.ws.Queue(channel)
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpStack create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpStack(w http.ResponseWriter, r *http.Request) {
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		node.ws.Stack(channel)
	}))
	handler.ServeHTTP(w, r)
}

/**
* HttpRemove create a stack channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func HttpRemove(w http.ResponseWriter, r *http.Request) {
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		err = node.ws.Remove(channel)
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
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		err = node.ws.Subscribe(channel, username)
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
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		err = node.ws.Unsubscribe(channel, username)
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
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		_, err = node.ws.SendTo(to, ms)
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
	handler := applyAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		_, err = node.ws.Publish(channel, ms)
		if err != nil {
			response.HTTPError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}))
	handler.ServeHTTP(w, r)
}

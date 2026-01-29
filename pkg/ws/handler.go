package ws

import (
	"fmt"
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* HttpConnect connect to the server using the http
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Hub) HttpConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader(w, r)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if conn == nil {
		response.HTTPError(w, r, http.StatusInternalServerError, "Connection is nil")
		return
	}

	ctx := r.Context()
	_, err = s.Connect(conn, ctx)
	if err != nil {
		logs.Alert(err)
	}
}

/**
* HttpTopic create a topic channel
* @param w http.ResponseWriter
* @param r *http.Request
**/
func (s *Hub) HttpTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := ctx.Value("username").(string)
	if username == "" {
		response.HTTPError(w, r, http.StatusUnauthorized, "Unauthorized")
		return
	}

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

	s.Topic(channel)
}

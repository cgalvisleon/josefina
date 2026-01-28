package ws

import (
	"net/http"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
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

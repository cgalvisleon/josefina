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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		response.HTTPError(w, r, http.StatusInternalServerError, err.Error())
	}

	_, err = s.connect(conn, "usuario")
	if err != nil {
		logs.Alert(err)
	}
}

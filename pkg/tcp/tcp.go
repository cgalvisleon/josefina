package tcp

import "github.com/cgalvisleon/et/tcp"

var (
	srv *tcp.Server
)

func New(port int) *tcp.Server {
	if srv == nil {
		srv = tcp.New(port)
	}

	return srv
}

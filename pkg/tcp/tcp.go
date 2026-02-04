package tcp

import "github.com/cgalvisleon/et/tcp"

var (
	srv *tcp.Server
)

/**
* New
* @param port int
* @return *tcp.Server
**/
func New(port int) *tcp.Server {
	if srv == nil {
		srv = tcp.New(port)
	}

	return srv
}

/**
* Init
* @return error
**/
func Init() error {
	return srv.Listen()
}

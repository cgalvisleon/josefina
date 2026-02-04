package tcp

import (
	"io"
	"net"

	"github.com/cgalvisleon/et/logs"
)

func (s *Server) handleServer(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		data := buf[:n]
		logs.Log("TCP", "Recibido:", string(data))

		conn.Write([]byte("ACK: "))
		conn.Write(data)
	}
}

func (s *Server) handleBalancer(client net.Conn) {
	defer client.Close()

	node := s.b.Next()
	if node == nil {
		return
	}

	backend, err := net.Dial("tcp", node.Address)
	if err != nil {
		return
	}
	defer backend.Close()

	node.Conns.Add(1)
	defer node.Conns.Add(-1)

	go io.Copy(backend, client)
	io.Copy(client, backend)
}

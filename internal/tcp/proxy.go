package tcp

import (
	"io"
	"net"
)

func Handle(client net.Conn, b *Balancer) {
	defer client.Close()

	node := b.Next()
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

	// Copia bidireccional
	go io.Copy(backend, client)
	io.Copy(client, backend)
}

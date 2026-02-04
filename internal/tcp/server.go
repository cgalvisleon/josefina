package tcp

type Server struct {
	port  int
	nodes []*Node
	b     *Balancer
}

func NewServer(port int) *Server {
	return &Server{
		port:  port,
		nodes: []*Node{},
	}
}

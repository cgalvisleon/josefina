package tcp

func New(port int) *Server {
	result := &Server{
		port:  port,
		nodes: []*Node{},
	}
	result.mode.Store(Follower)
	return result
}

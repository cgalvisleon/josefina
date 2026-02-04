package tcp

type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

type Server struct {
	port  int
	nodes []*Node
	b     *Balancer
	state NodeState
}

func NewServer(port int) *Server {
	return &Server{
		port:  port,
		nodes: []*Node{},
	}
}

func (s *Server) Start(state NodeState) error {
	s.nodes = append(s.nodes, node)
}

package rds

type Follow struct{}

func (s *Follow) GetNode() string {
	return node.Host
}

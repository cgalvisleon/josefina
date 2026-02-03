package core

type Core struct {
	getLeader func() (string, bool)
	address   string
}

var (
	syn *Core
)

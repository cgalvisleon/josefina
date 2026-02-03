package core

type Core struct {
	getLeader func() (string, bool)
}

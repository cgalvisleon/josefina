package rds

import "sync"

type Votes struct {
	votes map[string]map[string]int
	mu    sync.Mutex
}

var votes *Votes

func init() {
	votes = &Votes{
		votes: make(map[string]map[string]int),
		mu:    sync.Mutex{},
	}
}

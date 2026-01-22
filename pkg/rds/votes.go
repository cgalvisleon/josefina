package rds

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Votes struct {
	votes map[string]string
	mu    sync.Mutex
}

var (
	votes *Votes
)

func init() {
	votes = &Votes{
		votes: make(map[string]string),
		mu:    sync.Mutex{},
	}
}

/**
* getVote: Returns the votes for a tag
* @param tag string
* @return string, error
**/
func getVote(tag, host string) (string, error) {
	if methods == nil {
		return "", fmt.Errorf(msg.MSG_METHODS_NOT_INITIALIZED)
	}

	votes.mu.Lock()
	defer votes.mu.Unlock()

	result, ok := votes.votes[tag]
	if ok {
		time.AfterFunc(1*time.Minute, func() {
			delete(votes.votes, tag)
		})

		return result, nil
	}

	votes.votes[tag] = host
	nodes, err := getNodes()
	if err != nil {
		return "", err
	}

	for _, host := range nodes {
		methods.getVote(tag, host)
	}

	return result, nil
}

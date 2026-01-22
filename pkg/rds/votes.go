package rds

import (
	"fmt"
	"sync"

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
* makeVote: Returns the votes for a tag
* @param tag string
* @return string, error
**/
func makeVote(tag string) (string, error) {
	if node == nil {
		return "", fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	if methods == nil {
		return "", fmt.Errorf(msg.MSG_METHODS_NOT_INITIALIZED)
	}

	votes.mu.Lock()
	defer votes.mu.Unlock()

	result, ok := votes.votes[tag]
	if ok {
		return result, nil
	}

	votes.votes[tag] = node.host
	nodes, err := getNodes()
	if err != nil {
		return "", err
	}

	results := make(map[string]int)
	results[node.host]++
	post := -1
	for i, host := range nodes {
		if host == node.host {
			post = i
			continue
		}

		res, err := methods.getVote(tag, host)
		if err != nil {
			continue
		}
		results[res]++
	}

	maxVotos := -1
	for host, v := range results {
		if v > maxVotos {
			maxVotos = v
			result = host
		}
	}

	return result, nil
}

/**
* vote: Returns the votes for a tag
* @param tag string
* @return string, error
**/
func vote(tag, host string) string {
	votes.mu.Lock()
	defer votes.mu.Unlock()

	_, ok := votes.votes[tag]
	if !ok {
		votes.votes[tag] = host
	}

	return votes.votes[tag]
}

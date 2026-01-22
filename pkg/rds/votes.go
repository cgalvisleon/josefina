package rds

import (
	"fmt"
	"math/rand"
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

func randomElectionTimeout(min, max time.Duration) time.Duration {
	// min y max deben ser > 0 y max > min
	delta := max - min
	return min + time.Duration(rand.Int63n(int64(delta)))
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

	heartbeat := 100 * time.Millisecond
	minElection := 10 * heartbeat // 1s
	maxElection := 20 * heartbeat // 2s

	timeout := randomElectionTimeout(minElection, maxElection)
	time.Sleep(timeout)

	go vote(tag, node.host)

	nodes, err := getNodes()
	if err != nil {
		return "", err
	}

	if len(nodes) < 2 {
		return node.host, nil
	}

	for _, host := range nodes {
		if host == node.host {
			continue
		}

		methods.vote(host, tag, node.host)
	}

	results := make(map[string]int)
	results[node.host]++
	for _, host := range nodes {
		if host == node.host {
			continue
		}

		res, err := methods.getVote(host, tag)
		if err != nil {
			continue
		}
		results[res]++
	}

	result := ""
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
func vote(tag, host string) {
	votes.mu.Lock()
	_, ok := votes.votes[tag]
	if !ok {
		votes.votes[tag] = host
	}
	votes.mu.Unlock()
}

/**
* vote: Returns the votes for a tag
* @param tag string
* @return string, error
**/
func getVote(tag string) string {
	votes.mu.Lock()
	result, ok := votes.votes[tag]
	if !ok {
		result = ""
	}
	votes.mu.Unlock()

	return result
}

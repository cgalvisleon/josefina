package jdb

import (
	"math/rand"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
)

var (
	rngMu sync.Mutex
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

/**
* randomBetween
* @param minMs, maxMs int
* @return time.Duration
**/
func randomBetween(minMs, maxMs int) time.Duration {
	if minMs >= maxMs {
		return time.Duration(minMs) * time.Millisecond
	}

	rngMu.Lock()
	n := rng.Intn(maxMs-minMs+1) + minMs
	rngMu.Unlock()

	return time.Duration(n) * time.Millisecond
}

/**
* majority
* @param n int
* @return int
**/
func majority(n int) int {
	return (n / 2) + 1
}

type RequestVoteArgs struct {
	Term        int
	CandidateID string
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type HeartbeatArgs struct {
	Term     int
	LeaderID string
}

type HeartbeatReply struct {
	Term int
	Ok   bool
}

/**
* electionLoop
**/
func (n *Node) electionLoop() {
	for {
		timeout := randomBetween(1500, 3000)
		time.Sleep(100 * time.Millisecond)

		n.mu.Lock()
		if n.state == Leader {
			n.mu.Unlock()
			continue
		}

		elapsed := time.Since(n.lastHeartbeat)
		n.mu.Unlock()

		if elapsed >= timeout {
			n.startElection()
		}
	}
}

/**
* startElection
**/
func (n *Node) startElection() {
	n.mu.Lock()
	n.state = Candidate
	n.term++
	term := n.term
	n.votedFor = n.host
	votes := 1
	n.mu.Unlock()

	for _, peer := range n.peers {
		go func(peer string) {
			args := RequestVoteArgs{Term: term, CandidateID: n.host}
			var reply RequestVoteReply
			ok := methods.requestVote(peer, &args, &reply)
			if ok {
				n.mu.Lock()
				defer n.mu.Unlock()

				if reply.Term > n.term {
					n.term = reply.Term
					n.state = Follower
					n.votedFor = ""
					return
				}

				if n.state == Candidate && reply.VoteGranted && term == n.term {
					votes++
					needed := majority(len(n.peers) + 1) // +1 porque tú eres un nodo
					if votes >= needed {
						n.becomeLeader()
					}
				}
			}
		}(peer)
	}
}

/**
* becomeLeader
**/
func (n *Node) becomeLeader() {
	n.mu.Lock()
	n.state = Leader
	n.leaderID = n.host
	n.lastHeartbeat = time.Now()
	n.mu.Unlock()

	go n.heartbeatLoop()
}

/**
* heartbeatLoop
**/
func (n *Node) heartbeatLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		n.mu.Lock()
		if n.state != Leader {
			n.mu.Unlock()
			return
		}
		term := n.term
		n.mu.Unlock()

		for _, peer := range n.peers {
			go func(peer string) {
				args := HeartbeatArgs{Term: term, LeaderID: n.host}
				var reply HeartbeatReply
				ok := methods.heartbeat(peer, &args, &reply)
				if ok {
					n.mu.Lock()
					defer n.mu.Unlock()

					if reply.Term > n.term {
						n.term = reply.Term
						n.state = Follower
						n.votedFor = ""
					}
				}
			}(peer)
		}
	}
}

/**
* requestVote
* @param args *RequestVoteArgs, reply *RequestVoteReply
* @return error
**/
func (n *Node) requestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if args.Term < n.term {
		reply.Term = n.term
		reply.VoteGranted = false
		return nil
	}

	if args.Term > n.term {
		n.term = args.Term
		n.state = Follower
		n.votedFor = ""
	}

	if n.votedFor == "" || n.votedFor == args.CandidateID {
		n.votedFor = args.CandidateID
		reply.VoteGranted = true

		//reinicia el timer de elección
		n.lastHeartbeat = time.Now()
	} else {
		reply.VoteGranted = false
	}

	reply.Term = n.term
	return nil
}

/**
* heartbeat
* @param args *HeartbeatArgs, reply *HeartbeatReply
* @return error
**/
func (n *Node) heartbeat(args *HeartbeatArgs, reply *HeartbeatReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if args.Term < n.term {
		reply.Term = n.term
		reply.Ok = false
		return nil
	}

	if args.Term > n.term {
		n.term = args.Term
		n.votedFor = ""
	}

	if args.LeaderID != n.host {
		n.state = Follower
	}

	n.leaderID = args.LeaderID
	n.lastHeartbeat = time.Now()

	if n.host == n.leaderID {
		logs.Log("Raft", "[", n.host, "] I am now the leader vote:", n.term)
	}

	reply.Term = n.term
	reply.Ok = true
	return nil
}

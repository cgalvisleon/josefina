package jdb

import (
	"math/rand"
	"sync"
	"time"
)

var (
	rngMu             sync.Mutex
	rng               = rand.New(rand.NewSource(time.Now().UnixNano()))
	heartbeatInterval = 250 * time.Millisecond
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

type ResponseBool struct {
	Ok    bool
	Error error
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
		time.Sleep(timeout)

		n.mu.Lock()
		elapsed := time.Since(n.lastHeartbeat)
		state := n.state
		n.mu.Unlock()

		if n.leaderID == "" && state != Leader && elapsed > timeout {
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

	total := len(n.peers) + 1 // +1 porque tú eres un nodo
	for _, peer := range n.peers {
		go func(peer string) {
			args := RequestVoteArgs{Term: term, CandidateID: n.host}
			var reply RequestVoteReply
			res := methods.requestVote(peer, &args, &reply)
			if res.Error != nil {
				total--
			}
			if res.Ok {
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
					needed := majority(total)
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
	ticker := time.NewTicker(heartbeatInterval)
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
				res := methods.heartbeat(peer, &args, &reply)
				if res.Ok {
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

	n.state = Follower
	n.leaderID = args.LeaderID
	n.lastHeartbeat = time.Now()

	reply.Term = n.term
	reply.Ok = true
	return nil
}

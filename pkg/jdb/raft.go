package jdb

import (
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/internal/catalog"
)

var (
	rngMu             sync.Mutex
	rng               = rand.New(rand.NewSource(time.Now().UnixNano()))
	heartbeatInterval = 500 * time.Millisecond
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
* getLeader
* @return string, error
**/
func (n *Node) getLeader() (string, bool) {
	n.mu.RLock()
	inCluster := n.inCluster
	result := n.leaderID
	n.mu.RUnlock()
	if !inCluster {
		return "", false
	}
	return result, result != n.Address && result != ""
}

/**
* electionLoop
**/
func (s *Node) electionLoop() {
	s.mu.Lock()
	s.state = Follower
	s.inCluster = len(s.peers) > 1
	s.lastHeartbeat = timezone.Now()
	s.mu.Unlock()

	for {
		timeout := randomBetween(1500, 3000)
		time.Sleep(timeout)

		s.mu.RLock()
		elapsed := time.Since(s.lastHeartbeat)
		state := s.state
		s.mu.RUnlock()

		if elapsed > heartbeatInterval && state != Leader {
			s.startElection()
		}
	}
}

/**
* startElection
**/
func (s *Node) startElection() {
	idx := slices.Index(s.peers, s.Address)
	if idx == -1 {
		s.mu.Lock()
		s.inCluster = false
		s.becomeLeader()
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.inCluster = true
	s.state = Candidate
	s.term++
	term := s.term
	s.votedFor = s.Address
	s.mu.Unlock()

	votes := 1
	total := len(s.peers)
	for _, peer := range s.peers {
		go func(peer string) {
			args := RequestVoteArgs{Term: term, CandidateID: s.Address}
			var reply RequestVoteReply
			res := requestVote(peer, &args, &reply)
			if res.Error != nil {
				total--
			}

			if res.Ok {
				s.mu.Lock()
				defer s.mu.Unlock()

				if reply.Term > s.term {
					s.term = reply.Term
					s.state = Follower
					s.votedFor = ""
					return
				}

				if s.state == Candidate && reply.VoteGranted && term == s.term {
					votes++
					needed := majority(total)
					if votes >= needed {
						s.becomeLeader()
					}
				}
			}
		}(peer)
	}
}

/**
* becomeLeader
**/
func (s *Node) becomeLeader() {
	s.state = Leader
	s.leaderID = s.Address
	s.lastHeartbeat = timezone.Now()

	logs.Logf(s.PackageName, "I am leader %s", s.Address)

	go s.heartbeatLoop()
}

/**
* heartbeatLoop
**/
func (s *Node) heartbeatLoop() {
	if !s.inCluster {
		return
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		state := s.state
		term := s.term
		s.mu.RUnlock()
		if state != Leader {
			return
		}

		for _, peer := range s.peers {
			if peer == s.Address {
				continue
			}

			go func(peer string) {
				args := HeartbeatArgs{Term: term, LeaderID: s.Address}
				var reply HeartbeatReply
				res := heartbeat(peer, &args, &reply)
				if res.Ok {
					s.mu.Lock()
					defer s.mu.Unlock()

					if reply.Term > s.term {
						s.term = reply.Term
						s.state = Follower
						s.votedFor = ""
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
func (s *Node) requestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.Term < s.term {
		reply.Term = s.term
		reply.VoteGranted = false
		return nil
	}

	if args.Term > s.term {
		s.term = args.Term
		s.state = Follower
		s.votedFor = ""
	}

	if s.votedFor == "" || s.votedFor == args.CandidateID {
		s.votedFor = args.CandidateID
		reply.VoteGranted = true
		s.lastHeartbeat = timezone.Now()
	} else {
		reply.VoteGranted = false
	}

	reply.Term = s.term
	return nil
}

/**
* heartbeat
* @param args *HeartbeatArgs, reply *HeartbeatReply
* @return error
**/
func (s *Node) heartbeat(args *HeartbeatArgs, reply *HeartbeatReply) error {
	changedLeader := false

	s.mu.Lock()

	if args.Term < s.term {
		reply.Term = s.term
		reply.Ok = false
		s.mu.Unlock()
		return nil
	}

	if args.Term > s.term {
		s.term = args.Term
		s.votedFor = ""
	}

	oldLeader := s.leaderID
	s.state = Follower
	s.leaderID = args.LeaderID
	s.lastHeartbeat = timezone.Now()

	if oldLeader != args.LeaderID {
		logs.Logf(appName, "Set leader %s in %s", args.LeaderID, s.Address)
		changedLeader = true
	}

	reply.Term = s.term
	reply.Ok = true
	s.mu.Unlock()

	if changedLeader {
		s.onChangeLeader()
	}
	return nil
}

/**
* onChangeLeader
**/
func (s *Node) onChangeLeader() {
	s.modelMu.RLock()
	models := make(map[string]*catalog.Model, len(s.models))
	for k, v := range s.models {
		models[k] = v
	}
	s.modelMu.RUnlock()

	err := s.reportModels(models)
	if err != nil {
		logs.Errorf("onChangeLeader: %s", err)
	}
}

/**
* requestVote
* @param to string, require *RequestVoteArgs, response *RequestVoteReply
* @return *ResponseBool
**/
func requestVote(to string, require *RequestVoteArgs, response *RequestVoteReply) *ResponseBool {
	var res RequestVoteReply
	err := jrpc.CallRpc(to, "Node.RequestVote", require, &res)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	*response = res
	return &ResponseBool{
		Ok:    true,
		Error: nil,
	}
}

/**
* RequestVote: Requests a vote
* @param require *RequestVoteArgs, response *RequestVoteReply
* @return error
**/
func (s *Node) RequestVote(require *RequestVoteArgs, response *RequestVoteReply) error {
	err := s.requestVote(require, response)
	return err
}

/**
* heartbeat: Sends a heartbeat
* @param require *HeartbeatArgs, response *HeartbeatReply
* @return error
**/
func heartbeat(to string, require *HeartbeatArgs, response *HeartbeatReply) *ResponseBool {
	var res HeartbeatReply
	err := jrpc.CallRpc(to, "Node.Heartbeat", require, &res)
	if err != nil {
		return &ResponseBool{
			Ok:    false,
			Error: err,
		}
	}

	*response = res
	return &ResponseBool{
		Ok:    true,
		Error: nil,
	}
}

/**
* Heartbeat: Sends a heartbeat
* @param require *HeartbeatArgs, response *HeartbeatReply
* @return error
**/
func (s *Node) Heartbeat(require *HeartbeatArgs, response *HeartbeatReply) error {
	err := s.heartbeat(require, response)
	return err
}

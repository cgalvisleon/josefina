package jdb

import (
	"math/rand"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
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
* electionLoop
**/
func (s *Node) electionLoop() {
	for {
		timeout := randomBetween(1500, 3000)
		time.Sleep(timeout)

		s.mu.Lock()
		elapsed := time.Since(s.lastHeartbeat)
		state := s.state
		s.mu.Unlock()

		if elapsed > heartbeatInterval && state != Leader {
			s.startElection()
		}
	}
}

/**
* startElection
**/
func (s *Node) startElection() {
	s.mu.Lock()
	s.state = Candidate
	s.term++
	term := s.term
	s.votedFor = s.Host
	votes := 1
	s.mu.Unlock()

	total := len(s.peers)
	for _, peer := range s.peers {
		go func(peer string) {
			args := RequestVoteArgs{Term: term, CandidateID: s.Host}
			var reply RequestVoteReply
			res := methods.requestVote(peer, &args, &reply)
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
	s.leaderID = s.Host
	s.lastHeartbeat = timezone.Now()

	logs.Debugf("I am leader %s", s.Host)

	go s.heartbeatLoop()
}

/**
* heartbeatLoop
**/
func (s *Node) heartbeatLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		if s.state != Leader {
			s.mu.Unlock()
			return
		}
		term := s.term
		s.mu.Unlock()

		for _, peer := range s.peers {
			if peer == s.Host {
				continue
			}

			go func(peer string) {
				args := HeartbeatArgs{Term: term, LeaderID: s.Host}
				var reply HeartbeatReply
				res := methods.heartbeat(peer, &args, &reply)
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if args.Term < s.term {
		reply.Term = s.term
		reply.Ok = false
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
		logs.Logf("Set leader %s in %s", args.LeaderID, s.Host)
		s.onChangeLeader()
	}

	reply.Term = s.term
	reply.Ok = true
	return nil
}

/**
* onChangeLeader
**/
func (s *Node) onChangeLeader() {
	err := s.reportModels(s.models)
	if err != nil {
		logs.Errorf("onChangeLeader: %s", err)
	}
}

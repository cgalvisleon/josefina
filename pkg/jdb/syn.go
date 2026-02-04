package jdb

import (
	"encoding/gob"
	"errors"
	"time"

	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/internal/msg"
)

func init() {
	gob.Register(time.Time{})
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(claim.Claim{})
	gob.Register(core.Session{})
	gob.Register(RequestVoteArgs{})
	gob.Register(RequestVoteReply{})
	gob.Register(HeartbeatArgs{})
	gob.Register(HeartbeatReply{})
	gob.Register(mem.Item{})
}

type AnyResult struct {
	Dest any
	Ok   bool
}

type Nodes struct{}

var syn *Nodes

func init() {
	syn = &Nodes{}
}

/**
* requestVote
* @param require et.Json, response *Model
* @return error
**/
func (s *Nodes) requestVote(to string, require *RequestVoteArgs, response *RequestVoteReply) *ResponseBool {
	var res RequestVoteReply
	err := jrpc.CallRpc(to, "Nodes.RequestVote", require, &res)
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
func (s *Nodes) RequestVote(require *RequestVoteArgs, response *RequestVoteReply) error {
	err := node.requestVote(require, response)
	return err
}

/**
* heartbeat: Sends a heartbeat
* @param require *HeartbeatArgs, response *HeartbeatReply
* @return error
**/
func (s *Nodes) heartbeat(to string, require *HeartbeatArgs, response *HeartbeatReply) *ResponseBool {
	var res HeartbeatReply
	err := jrpc.CallRpc(to, "Nodes.Heartbeat", require, &res)
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
func (s *Nodes) Heartbeat(require *HeartbeatArgs, response *HeartbeatReply) error {
	err := node.heartbeat(require, response)
	return err
}

/**
* onConnect: Handles a connection
* @param to, idx string, dest any
* @return error
**/
func (s *Nodes) onConnect(to string, username string, tpConnection TpConnection, host string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"username":     username,
		"tpConnection": tpConnection,
		"host":         host,
	}
	var dest bool
	err := jrpc.CallRpc(to, "Nodes.OnConnect", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* OnConnect: Handles a connection
* @param require et.Json, response *boolean
* @return error
**/
func (s *Nodes) OnConnect(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	tpConnection := TpConnection(require.Int("tpConnection"))
	host := require.Str("host")
	err := node.onConnect(username, tpConnection, host)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* onDisconnect: Handles a disconnection
* @param to, idx string, dest any
* @return error
**/
func (s *Nodes) onDisconnect(to string, username string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"username": username,
	}
	var dest bool
	err := jrpc.CallRpc(to, "Nodes.OnDisconnect", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* OnDisconnect: Handles a disconnection
* @param require et.Json, response *boolean
* @return error
**/
func (s *Nodes) OnDisconnect(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	err := node.onDisconnect(username)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

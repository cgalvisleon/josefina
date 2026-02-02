package jdb

import (
	"encoding/gob"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/dbs"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

func init() {
	gob.Register(time.Time{})
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(&Session{})
	gob.Register(&RequestVoteArgs{})
	gob.Register(&RequestVoteReply{})
	gob.Register(&HeartbeatArgs{})
	gob.Register(&HeartbeatReply{})
	gob.Register(&mem.Item{})
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
* ping
* @return error
**/
func (s *Nodes) ping(to string) error {
	var response string
	err := jrpc.CallRpc(to, "Nodes.Ping", node.Host, &response)
	if err != nil {
		return err
	}

	logs.Logf(node.PackageName, "%s:%s", response, to)
	return nil
}

/**
* Ping: Pings the leader
* @param response *string
* @return error
**/
func (s *Nodes) Ping(require string, response *string) error {
	logs.Log(node.PackageName, "ping:", require)
	*response = "pong"
	return nil
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
* getDb: Gets a database
* @param to string, name string
* @return *DB, error
**/
func (s *Nodes) getDb(to string, name string) (*dbs.DB, error) {
	var response *dbs.DB
	err := jrpc.CallRpc(to, "Nodes.GetDb", name, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* GetDb: Gets a database
* @param require string, response *DB
* @return error
**/
func (s *Nodes) GetDb(require string, response *dbs.DB) error {
	db, err := node.getDb(require)
	if err != nil {
		return err
	}

	*response = *db
	return nil
}

/**
* setDb: Saves a database
* @param to string, db *DB
* @return error
**/
func (s *Nodes) setDb(to string, db *dbs.DB) error {
	var response bool
	err := jrpc.CallRpc(to, "Nodes.SetDb", db, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetDb: Saves a database
* @param model *DB
* @return bool, error
**/
func (s *Nodes) SetDb(require *dbs.DB, response *bool) error {
	err := node.setDb(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* getModel: Gets a model
* @param to string, database, schema, model string
* @return *Model, error
**/
func (s *Nodes) getModel(to, database, schema, name string) (*dbs.Model, error) {
	var response *dbs.Model
	err := jrpc.CallRpc(to, "Nodes.GetModel", et.Json{
		"database": database,
		"schema":   schema,
		"name":     name,
	}, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

/**
* GetModel: Gets a model
* @param require et.Json, response *Model
* @return error
**/
func (s *Nodes) GetModel(require et.Json, response *dbs.Model) error {
	database := require.Str("database")
	schema := require.Str("schema")
	name := require.Str("name")
	result, err := node.getModel(database, schema, name)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* loadModel: Loads a model
* @param to string, model *Model
* @return error
**/
func (s *Nodes) loadModel(to string, model *dbs.Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Nodes.LoadModel", model, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* LoadModel: Loads a model
* @param require *Model, response true
* @return error
**/
func (s *Nodes) LoadModel(require *dbs.Model, response *bool) error {
	err := node.loadModel(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* setModel: Saves a model
* @param to string, model *Model
* @return error
**/
func (s *Nodes) setModel(to string, model *dbs.Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Nodes.SetModel", model, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetModel: Saves a model
* @param model *Model
* @return bool, error
**/
func (s *Nodes) SetModel(require *dbs.Model, response *bool) error {
	err := node.setModel(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* reportModels: Reports models
* @param to string, models map[string]*Model
* @return error
**/
func (s *Nodes) reportModels(to string, models map[string]*dbs.Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Nodes.ReportModels", models, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* ReportModels: Reports models
* @param require map[string]*Model, response true
* @return error
**/
func (s *Nodes) ReportModels(require map[string]*dbs.Model, response *bool) error {
	err := node.reportModels(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* setTransaction: Sets a transaction
* @param to, key string, data et.Json
* @return error
**/
func (s *Nodes) setTransaction(to, key string, data et.Json) error {
	args := et.Json{
		"key":  key,
		"data": data,
	}
	var reply bool
	err := jrpc.CallRpc(to, "Nodes.SetTransaction", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetTransaction: Sets a transaction
* @param require et.Json, response *bool
* @return error
**/
func (s *Nodes) SetTransaction(require et.Json, response *bool) error {
	key := require.Str("key")
	data := require.Json("data")
	err := node.setTransaction(key, data)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* auth: Authenticates a user
* @param to, device, database, username, password string
* @return error
**/
func (s *Nodes) auth(to, device, database, username, password string) (*Session, error) {
	args := et.Json{
		"device":   device,
		"database": database,
		"username": username,
		"password": password,
	}
	var reply *Session
	err := jrpc.CallRpc(to, "Nodes.Auth", args, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* SetTransaction: Sets a transaction
* @param require et.Json, response *bool
* @return error
**/
func (s *Nodes) Auth(require et.Json, response *Session) error {
	device := require.Str("device")
	database := require.Str("database")
	username := require.Str("username")
	password := require.Str("password")
	result, err := node.auth(device, database, username, password)
	if err != nil {
		return err
	}

	response = result
	return nil
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

package node

import (
	"encoding/gob"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/jdb"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

func init() {
	gob.Register(time.Time{})
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(&jdb.DB{})
	gob.Register(&jdb.Schema{})
	gob.Register(&jdb.Model{})
	gob.Register(&Session{})
	gob.Register(&jdb.Tx{})
	gob.Register(&jdb.Transaction{})
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

type Syn struct{}

var syn *Syn

func init() {
	syn = &Syn{}
}

/**
* ping
* @return error
**/
func (s *Syn) ping(to string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var response string
	err := jrpc.CallRpc(to, "Syn.Ping", node.Host, &response)
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
func (s *Syn) Ping(require string, response *string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	logs.Log(node.PackageName, "ping:", require)
	*response = "pong"
	return nil
}

/**
* requestVote
* @param require et.Json, response *Model
* @return error
**/
func (s *Syn) requestVote(to string, require *RequestVoteArgs, response *RequestVoteReply) *ResponseBool {
	var res RequestVoteReply
	err := jrpc.CallRpc(to, "Syn.RequestVote", require, &res)
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
func (s *Syn) RequestVote(require *RequestVoteArgs, response *RequestVoteReply) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.requestVote(require, response)
	return err
}

/**
* heartbeat: Sends a heartbeat
* @param require *HeartbeatArgs, response *HeartbeatReply
* @return error
**/
func (s *Syn) heartbeat(to string, require *HeartbeatArgs, response *HeartbeatReply) *ResponseBool {
	var res HeartbeatReply
	err := jrpc.CallRpc(to, "Syn.Heartbeat", require, &res)
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
func (s *Syn) Heartbeat(require *HeartbeatArgs, response *HeartbeatReply) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.heartbeat(require, response)
	return err
}

/**
* getDb: Gets a database
* @param to string, name string
* @return *DB, error
**/
func (s *Syn) getDb(to string, name string) (*DB, error) {
	var response *DB
	err := jrpc.CallRpc(to, "Syn.GetDb", name, &response)
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
func (s *Syn) GetDb(require string, response *DB) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	db, err := getDb(require)
	if err != nil {
		return err
	}

	*response = *db
	return nil
}

/**
* getModel: Gets a model
* @param to string, database, schema, model string
* @return *Model, error
**/
func (s *Syn) getModel(to, database, schema, name string) (*Model, error) {
	var response Model
	err := jrpc.CallRpc(to, "Syn.GetModel", et.Json{
		"database": database,
		"schema":   schema,
		"name":     name,
	}, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

/**
* GetModel: Gets a model
* @param require et.Json, response *Model
* @return error
**/
func (s *Syn) GetModel(require et.Json, response *Model) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

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
func (s *Syn) loadModel(to string, model *Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.LoadModel", model, &response)
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
func (s *Syn) LoadModel(require *Model, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.loadModel(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* saveModel: Saves a model
* @param to string, model *Model
* @return error
**/
func (s *Syn) saveModel(to string, model *Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.SaveModel", model, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* SaveModel: Saves a model
* @param model *Model
* @return bool, error
**/
func (s *Syn) SaveModel(require *Model, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.saveModel(require)
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
func (s *Syn) reportModels(to string, models map[string]*Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.ReportModels", models, &response)
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
func (s *Syn) ReportModels(require map[string]*Model, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.reportModels(require)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* saveDb: Saves a database
* @param to string, db *DB
* @return error
**/
func (s *Syn) saveDb(to string, db *DB) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.SaveDb", db, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* SaveDb: Saves a database
* @param model *DB
* @return bool, error
**/
func (s *Syn) SaveDb(require *DB, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.saveDb(require)
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
func (s *Syn) setTransaction(to, key string, data et.Json) (string, error) {
	if node == nil {
		return "", errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"key":  key,
		"data": data,
	}
	var reply string
	err := jrpc.CallRpc(to, "Syn.SetTransaction", args, &reply)
	if err != nil {
		return "", err
	}

	return reply, nil
}

/**
* SetTransaction: Sets a transaction
* @param require et.Json, response *string
* @return error
**/
func (s *Syn) SetTransaction(require et.Json, response *string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	key := require.Str("key")
	data := require.Json("data")
	result, err := setTransaction(key, data)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

/**
* onConnect: Handles a connection
* @param to, idx string, dest any
* @return error
**/
func (s *Syn) onConnect(to string, username string, tpConnection TpConnection, host string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"username":     username,
		"tpConnection": tpConnection,
		"host":         host,
	}
	var dest bool
	err := jrpc.CallRpc(to, "Syn.OnConnect", args, &dest)
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
func (s *Syn) OnConnect(require et.Json, response *bool) error {
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
func (s *Syn) onDisconnect(to string, username string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"username": username,
	}
	var dest bool
	err := jrpc.CallRpc(to, "Syn.OnDisconnect", args, &dest)
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
func (s *Syn) OnDisconnect(require et.Json, response *bool) error {
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

package jdb

import (
	"encoding/gob"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

func init() {
	gob.Register(time.Time{})
	gob.Register(et.Json{})
	gob.Register([]et.Json{})
	gob.Register(et.Item{})
	gob.Register(et.Items{})
	gob.Register(et.List{})
	gob.Register(&DB{})
	gob.Register(&Schema{})
	gob.Register(&Model{})
	gob.Register(&Session{})
	gob.Register(&Tx{})
	gob.Register(&Transaction{})
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
* RequestVote
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
* RequestVote
* @param require et.Json, response *Model
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
* heartbeat
* @param require et.Json, response *Model
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
* HeartbeatHeartbeHeartbeatat
* @param require et.Json, response *Model
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
* getModel
* @param database, schema, model string
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
* GetFrom
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
* loadModel
* @param database, schema, model string
* @return *Model, error
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
* LoadModel
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
* saveModel
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
* SaveModel
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
* reportModels
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
* ReportModels
* @param model *Model
* @return bool, error
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
* getDb
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
* GetDb
* @param model *DB
* @return bool, error
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
* saveDb
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
* SaveDb
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
* signIn: Sign in a user
* @param to, username, password string
* @return *Session, error
**/
func (s *Syn) createUser(to, username, password string) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.CreateUser", et.Json{
		"username": username,
		"password": password,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* CreateUser
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) CreateUser(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	password := require.Str("password")
	err := createUser(username, password)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* signIn: Sign in a user
* @param device, username, password string
* @return error
**/
func (s *Syn) dropUser(to, username string) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.DropUser", et.Json{
		"username": username,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* getModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Syn) DropUser(require et.Json, response *Session) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	err := dropUser(username)
	if err != nil {
		return err
	}

	*response = Session{}
	return nil
}

/**
* changuePassword: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func (s *Syn) changuePassword(to, username, password string) error {
	var response bool
	err := jrpc.CallRpc(to, "Syn.ChanguePassword", et.Json{
		"username": username,
		"password": password,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* ChanguePassword
* @param database, schema, model string
* @return *Model, error
**/
func (s *Syn) ChanguePassword(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	username := require.Str("username")
	password := require.Str("password")
	err := changuePassword(username, password)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* auth: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func (s *Syn) auth(to, device, database, username, password string) (*Session, error) {
	var response Session
	err := jrpc.CallRpc(to, "Syn.Auth", et.Json{
		"device":   device,
		"database": database,
		"username": username,
		"password": password,
	}, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

/**
* Auth
* @param database, schema, model string
* @return *Model, error
**/
func (s *Syn) Auth(require et.Json, response *Session) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	device := require.Str("device")
	database := require.Str("database")
	username := require.Str("username")
	password := require.Str("password")
	result, err := Auth(device, database, username, password)
	if err != nil {
		return err
	}

	*response = *result
	return nil
}

/**
* createSerie
* @param to, tag, format string, value int
* @return error
**/
func (s *Syn) createSerie(to, tag, format string, value int) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag":    tag,
		"format": format,
		"value":  value,
	}
	var reply string
	err := jrpc.CallRpc(to, "Syn.CreateSerie", data, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* CreateSerie
* @param require *Transaction, response *Session
* @return error
**/
func (s *Syn) CreateSerie(require et.Json, response *string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	format := require.Str("format")
	value := require.Int("value")
	err := createSerie(tag, format, value)
	if err != nil {
		return err
	}

	return nil
}

/**
* dropSerie
* @param tag string
* @return error
**/
func (s *Syn) dropSerie(to, tag string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag": tag,
	}
	var reply string
	err := jrpc.CallRpc(to, "Syn.DropSerie", data, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* DropSerie
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) DropSerie(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	err := dropSerie(tag)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* setSerie
* @param to, tag string, value int
* @return error
**/
func (s *Syn) setSerie(to, tag string, value int) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag":   tag,
		"value": value,
	}
	var reply string
	err := jrpc.CallRpc(to, "Syn.SetSerie", data, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetSerie
* @param require et.Json, response *bool
* @return error
**/
func (s *Syn) SetSerie(require et.Json, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	value := require.Int("value")
	err := setSerie(tag, value)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* getSerie
* @param to, tag string
* @return error
**/
func (s *Syn) getSerie(to, tag string) (et.Json, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag": tag,
	}
	var reply et.Json
	err := jrpc.CallRpc(to, "Syn.GetSerie", data, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* GetSerie
* @param require et.Json, response *et.Json
* @return error
**/
func (s *Syn) GetSerie(require et.Json, response *et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	tag := require.Str("tag")
	result, err := getSerie(tag)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

/**
* setTransaction
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
* SetTransaction
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
* setCache
* @param to, key string, value interface{}, duration time.Duration
* @return error
**/
func (s *Syn) setCache(to, key string, value interface{}, duration time.Duration) (*mem.Item, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"key":      key,
		"value":    value,
		"duration": duration,
	}
	var reply *mem.Item
	err := jrpc.CallRpc(to, "Syn.SetCache", data, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* SetCache
* @param require et.Json, response *mem.Item
* @return error
**/
func (s *Syn) SetCache(require et.Json, response *mem.Item) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	key := require.Str("key")
	value := require.Str("value")
	duration := time.Duration(require.Int("duration"))
	result, err := SetCache(key, value, duration)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* getCache
* @param to, key string
* @return error
**/
func (s *Syn) getCache(to, key string) (*mem.Item, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var reply *mem.Item
	err := jrpc.CallRpc(to, "Syn.GetCache", key, &reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

/**
* GetCache
* @param require string, response *mem.Item
* @return error
**/
func (s *Syn) GetCache(require string, response *mem.Item) bool {
	if node == nil {
		return false
	}

	result, exists := GetCache(require)
	response = result
	return exists
}

/**
* deleteCache
* @param to, key string
* @return error
**/
func (s *Syn) deleteCache(to, key string) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var reply bool
	err := jrpc.CallRpc(to, "Syn.DeleteCache", key, &reply)
	if err != nil {
		return false, err
	}

	return reply, nil
}

/**
* DeleteCache
* @param require string, response *mem.Item
* @return error
**/
func (s *Syn) DeleteCache(require string, response *bool) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	result, err := DeleteCache(require)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

/**
* onConnect
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
* OnConnect
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
* onDisconnect
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
* OnDisconnect
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

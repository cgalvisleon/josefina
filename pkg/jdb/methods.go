package jdb

import (
	"encoding/gob"
	"fmt"
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

type Methods struct{}

var methods *Methods

/**
* ping
* @return error
**/
func (s *Methods) ping(to string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var response string
	err := jrpc.CallRpc(to, "Methods.Ping", node.host, &response)
	if err != nil {
		return err
	}

	logs.Logf(packageName, "%s:%s", response, to)
	return nil
}

/**
* Ping: Pings the leader
* @param response *string
* @return error
**/
func (s *Methods) Ping(require string, response *string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	logs.Log(packageName, "ping:", require)
	*response = "pong"
	return nil
}

/**
* RequestVote
* @param require et.Json, response *Model
* @return error
**/
func (s *Methods) requestVote(to string, require *RequestVoteArgs, response *RequestVoteReply) *ResponseBool {
	var res RequestVoteReply
	err := jrpc.CallRpc(to, "Methods.RequestVote", require, &res)
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
func (s *Methods) RequestVote(require *RequestVoteArgs, response *RequestVoteReply) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.requestVote(require, response)
	return err
}

/**
* heartbeat
* @param require et.Json, response *Model
* @return error
**/
func (s *Methods) heartbeat(to string, require *HeartbeatArgs, response *HeartbeatReply) *ResponseBool {
	var res HeartbeatReply
	err := jrpc.CallRpc(to, "Methods.Heartbeat", require, &res)
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
func (s *Methods) Heartbeat(require *HeartbeatArgs, response *HeartbeatReply) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	err := node.heartbeat(require, response)
	return err
}

/**
* getModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Methods) getModel(to, database, schema, name string) (*Model, error) {
	var response Model
	err := jrpc.CallRpc(to, "Methods.GetModel", et.Json{
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
func (s *Methods) GetModel(require et.Json, response *Model) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) loadModel(to string, model *Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Methods.ReserveModel", model, &response)
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
func (s *Methods) LoadModel(require *Model, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) saveModel(to string, model *Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Methods.SaveModel", model, &response)
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
func (s *Methods) SaveModel(require *Model, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) reportModels(to string, models map[string]*Model) error {
	var response bool
	err := jrpc.CallRpc(to, "Methods.ReportModels", models, &response)
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
func (s *Methods) ReportModels(require map[string]*Model, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) getDb(to string, name string) (*DB, error) {
	var response *DB
	err := jrpc.CallRpc(to, "Methods.GetDb", name, &response)
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
func (s *Methods) GetDb(require string, response *DB) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) saveDb(to string, db *DB) error {
	var response bool
	err := jrpc.CallRpc(to, "Methods.SaveDb", db, &response)
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
func (s *Methods) SaveDb(require *DB, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) createUser(to, username, password string) error {
	var response bool
	err := jrpc.CallRpc(to, "Methods.CreateUser", et.Json{
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
func (s *Methods) CreateUser(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) dropUser(to, username string) error {
	var response bool
	err := jrpc.CallRpc(to, "Methods.DropUser", et.Json{
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
func (s *Methods) DropUser(require et.Json, response *Session) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) changuePassword(to, username, password string) error {
	var response bool
	err := jrpc.CallRpc(to, "Methods.ChanguePassword", et.Json{
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
func (s *Methods) ChanguePassword(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
* signIn: Sign in a user
* @param device, username, password string
* @return *Session, error
**/
func (s *Methods) signIn(to, device, database, username, password string) (*Session, error) {
	var response Session
	err := jrpc.CallRpc(to, "Methods.SignIn", et.Json{
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
* getModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Methods) SignIn(require et.Json, response *Session) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	device := require.Str("device")
	database := require.Str("database")
	username := require.Str("username")
	password := require.Str("password")
	result, err := SignIn(device, database, username, password)
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
func (s *Methods) createSerie(to, tag, format string, value int) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag":    tag,
		"format": format,
		"value":  value,
	}
	var reply string
	err := jrpc.CallRpc(to, "Methods.CreateSerie", data, &reply)
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
func (s *Methods) CreateSerie(require et.Json, response *string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) dropSerie(to, tag string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag": tag,
	}
	var reply string
	err := jrpc.CallRpc(to, "Methods.DropSerie", data, &reply)
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
func (s *Methods) DropSerie(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) setSerie(to, tag string, value int) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag":   tag,
		"value": value,
	}
	var reply string
	err := jrpc.CallRpc(to, "Methods.SetSerie", data, &reply)
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
func (s *Methods) SetSerie(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) getSerie(to, tag string) (et.Json, error) {
	if node == nil {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"tag": tag,
	}
	var reply et.Json
	err := jrpc.CallRpc(to, "Methods.GetSerie", data, &reply)
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
func (s *Methods) GetSerie(require et.Json, response *et.Json) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) setTransaction(to, key string, data et.Json) (string, error) {
	if node == nil {
		return "", fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"key":  key,
		"data": data,
	}
	var reply string
	err := jrpc.CallRpc(to, "Methods.SetTransaction", args, &reply)
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
func (s *Methods) SetTransaction(require et.Json, response *string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) setCache(to, key string, value interface{}, duration time.Duration) (*mem.Item, error) {
	if node == nil {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"key":      key,
		"value":    value,
		"duration": duration,
	}
	var reply *mem.Item
	err := jrpc.CallRpc(to, "Methods.SetCache", data, &reply)
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
func (s *Methods) SetCache(require et.Json, response *mem.Item) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
func (s *Methods) getCache(to, key string) (*mem.Item, error) {
	if node == nil {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var reply *mem.Item
	err := jrpc.CallRpc(to, "Methods.GetCache", key, &reply)
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
func (s *Methods) GetCache(require string, response *mem.Item) bool {
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
func (s *Methods) deleteCache(to, key string) (bool, error) {
	if node == nil {
		return false, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var reply bool
	err := jrpc.CallRpc(to, "Methods.DeleteCache", key, &reply)
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
func (s *Methods) DeleteCache(require string, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	result, err := DeleteCache(require)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

/**
* put
* @param to, key string
* @return error
**/
func (s *Methods) put(from *From, idx string, data any) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
		"data": data,
	}
	var reply bool
	err := jrpc.CallRpc(from.Host, "Methods.Put", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* Put
* @param require et.Json, response *bool
* @return error
**/
func (s *Methods) Put(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	data := require.Get("data")
	err := Put(from, idx, data)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* remove
* @param to, key string
* @return error
**/
func (s *Methods) remove(from *From, idx string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var reply bool
	err := jrpc.CallRpc(from.Host, "Methods.Remove", args, &reply)
	if err != nil {
		return err
	}

	return nil
}

/**
* Remove
* @param require et.Json, response *bool
* @return error
**/
func (s *Methods) Remove(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	err := Remove(from, idx)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* get
* @param to, idx string, dest any
* @return error
**/
func (s *Methods) get(from *From, idx string, dest any) (bool, error) {
	if node == nil {
		return false, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var reply AnyResult
	err := jrpc.CallRpc(from.Host, "Methods.Get", args, &reply)
	if err != nil {
		return false, err
	}

	dest = reply.Dest
	return reply.Ok, nil
}

/**
* Get
* @param require et.Json, response *AnyResult
* @return error
**/
func (s *Methods) Get(require et.Json, response *AnyResult) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	var dest any
	ok, err := Get(from, idx, &dest)
	if err != nil {
		return err
	}

	*response = AnyResult{
		Dest: dest,
		Ok:   ok,
	}
	return nil
}

/**
* putObject
* @param to, idx string, dest any
* @return error
**/
func (s *Methods) putObject(from *From, idx string, dest et.Json) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	err := jrpc.CallRpc(from.Host, "Methods.PutObject", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* PutObject
* @param require et.Json, response et.Json
* @return error
**/
func (s *Methods) PutObject(require et.Json, response et.Json) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	var dest et.Json
	err := PutObject(from, idx, dest)
	if err != nil {
		return err
	}

	response = dest
	return nil
}

/**
* removeObject
* @param to, idx string, dest any
* @return error
**/
func (s *Methods) removeObject(from *From, idx string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from": from,
		"idx":  idx,
	}
	var dest bool
	err := jrpc.CallRpc(from.Host, "Methods.RemoveObject", args, &dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* RemoveObject
* @param require et.Json, response *bool
* @return error
**/
func (s *Methods) RemoveObject(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	idx := require.Str("idx")
	err := RemoveObject(from, idx)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* isExisted
* @param to, idx string, dest any
* @return error
**/
func (s *Methods) isExisted(from *From, field, idx string) (bool, error) {
	if node == nil {
		return false, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	args := et.Json{
		"from":  from,
		"field": field,
		"idx":   idx,
	}
	var dest bool
	err := jrpc.CallRpc(from.Host, "Methods.IsExisted", args, &dest)
	if err != nil {
		return false, err
	}

	return dest, nil
}

/**
* IsExisted
* @param require et.Json, response *bool
* @return error
**/
func (s *Methods) IsExisted(require et.Json, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	from := toFrom(require.Json("from"))
	field := require.Str("field")
	idx := require.Str("idx")
	existed, err := IsExisted(from, field, idx)
	if err != nil {
		return err
	}

	*response = existed
	return nil
}

/**
* count
* @param to, idx string, dest any
* @return error
**/
func (s *Methods) count(from *From) (int, error) {
	if node == nil {
		return 0, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	var dest int
	err := jrpc.CallRpc(from.Host, "Methods.Count", from, &dest)
	if err != nil {
		return 0, err
	}

	return dest, nil
}

/**
* Count
* @param require *From, response *int
* @return error
**/
func (s *Methods) Count(require *From, response *int) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	existed, err := Count(require)
	if err != nil {
		return err
	}

	*response = existed
	return nil
}

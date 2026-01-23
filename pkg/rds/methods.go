package rds

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrpc"
	"github.com/cgalvisleon/et/logs"
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
* vote
* @param tag, host string
* @return error
**/
func (s *Methods) vote(to, tag, host string) error {
	data := et.Json{
		"tag":  tag,
		"host": host,
	}
	var response string
	err := jrpc.CallRpc(to, "Methods.Vote", data, &response)
	if err != nil {
		return err
	}

	return nil
}

/**
* GetVote: Returns the votes for a tag
* @param require et.Json, response *string
* @return error
**/
func (s *Methods) Vote(require et.Json, response *string) error {
	tag := require.Str("tag")
	host := require.Str("host")
	go vote(tag, host)

	return nil
}

/**
* vote
* @param tag, host string
* @return error
**/
func (s *Methods) getVote(to, tag string) (string, error) {
	var response string
	err := jrpc.CallRpc(to, "Methods.GetVote", tag, &response)
	if err != nil {
		return "", err
	}

	return response, nil
}

/**
* GetVote: Returns the votes for a tag
* @param require string, response *string
* @return error
**/
func (s *Methods) GetVote(require string, response *string) error {
	tag := require
	result := make(chan string)
	go func() {
		res := getVote(tag)
		result <- res
	}()

	*response = <-result
	return nil
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
* getModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Methods) GetModel(require et.Json, response *Model) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	type modelResult struct {
		result *Model
		err    error
	}

	database := require.Str("database")
	schema := require.Str("schema")
	name := require.Str("name")

	ch := make(chan modelResult, 1)
	go func() {
		result, err := node.getModel(database, schema, name)
		ch <- modelResult{result: result, err: err}
	}()

	result := <-ch
	if result.err != nil {
		return result.err
	}

	*response = *result.result
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
* loadModel
* @param to string, model *Model
* @return error
**/
func (s *Methods) loadModel(to string, model *Model) (bool, error) {
	var response bool
	err := jrpc.CallRpc(to, "Methods.LoadModel", model, &response)
	if err != nil {
		return false, err
	}

	return response, nil
}

/**
* LoadModel
* @param model *Model
* @return bool, error
**/
func (s *Methods) LoadModel(require *Model, response *bool) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	type boolResult struct {
		result bool
		err    error
	}

	ch := make(chan boolResult, 1)
	go func() {
		result, err := node.loadModel(require)
		ch <- boolResult{result: result, err: err}
	}()

	result := <-ch
	if result.err != nil {
		return result.err
	}

	*response = result.result
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
	result, err := node.signIn(device, database, username, password)
	if err != nil {
		return err
	}

	*response = *result
	return nil
}

/**
* setRecord
* @param schema, model, key string
* @return error
**/
func (s *Methods) setRecord(to, schema, model, key string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"scgema": schema,
		"model":  model,
		"key":    key,
	}
	var reply string
	err := jrpc.CallRpc(to, "Methods.SetRecord", data, &reply)
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
func (s *Methods) SetRecord(require et.Json, response *string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	schema := require.Str("schema")
	model := require.Str("model")
	key := require.Str("key")
	err := setRecord(schema, model, key)
	if err != nil {
		return err
	}

	return nil
}

/**
* createSerie
* @param name, tag, format string, value int
* @return error
**/
func (s *Methods) createSerie(to, name, tag, format string, value int) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"name":   name,
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

	name := require.Str("name")
	tag := require.Str("tag")
	format := require.Str("format")
	value := require.Int("value")
	err := createSerie(name, tag, format, value)
	if err != nil {
		return err
	}

	return nil
}

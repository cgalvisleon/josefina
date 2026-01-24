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
* getFrom
* @param database, schema, model string
* @return *Model, error
**/
func (s *Methods) getFrom(to, database, schema, name string) (*From, error) {
	var response From
	err := jrpc.CallRpc(to, "Methods.GetFrom", et.Json{
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
* @param require et.Json, response *From
* @return error
**/
func (s *Methods) GetFrom(require et.Json, response *From) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	database := require.Str("database")
	schema := require.Str("schema")
	name := require.Str("name")
	result, err := node.getFrom(database, schema, name)
	if err != nil {
		return err
	}

	response = result
	return nil
}

/**
* reserveModel
* @param database, schema, model string
* @return *Model, error
**/
func (s *Methods) reserveModel(to string, from *From) (*Reserve, error) {
	var response Reserve
	err := jrpc.CallRpc(to, "Methods.ReserveModel", from, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

/**
* ReserveModel
* @param require *From, response *Reserve
* @return error
**/
func (s *Methods) ReserveModel(require *From, response *Reserve) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	result, err := node.reserveModel(require)
	if err != nil {
		return err
	}

	response = result
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
	err := node.createUser(username, password)
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
	err := node.dropUser(username)
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
	err := node.changuePassword(username, password)
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

/**
* dropSerie
* @param name, tag string
* @return error
**/
func (s *Methods) dropSerie(to, name, tag string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"name": name,
		"tag":  tag,
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

	name := require.Str("name")
	tag := require.Str("tag")
	err := dropSerie(name, tag)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* setSerie
* @param to, name, tag string, value int
* @return error
**/
func (s *Methods) setSerie(to, name, tag string, value int) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"name":  name,
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

	name := require.Str("name")
	tag := require.Str("tag")
	value := require.Int("value")
	err := setSerie(name, tag, value)
	if err != nil {
		return err
	}

	*response = true
	return nil
}

/**
* getSerie
* @param to, name, tag string, value int
* @return error
**/
func (s *Methods) getSerie(to, name, tag string) (et.Json, error) {
	if node == nil {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	data := et.Json{
		"name": name,
		"tag":  tag,
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

	name := require.Str("name")
	tag := require.Str("tag")
	result, err := getSerie(name, tag)
	if err != nil {
		return err
	}

	*response = result
	return nil
}

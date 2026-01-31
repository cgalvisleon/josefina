package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Host     string `json:"-"`
	IsInit   bool   `json:"-"`
}

/**
* key: Returns the key of the model
* @return string
**/
func (s *From) key() string {
	return modelKey(s.Database, s.Schema, s.Name)
}

/**
* toFrom: Converts a JSON to a From
* @param def et.Json
* @return *From
**/
func toFrom(def et.Json) *From {
	return &From{
		Database: def.Str("database"),
		Schema:   def.Str("schema"),
		Name:     def.Str("name"),
		Host:     def.Str("host"),
		IsInit:   def.Bool("is_init"),
	}
}

/**
* Put: Puts the model
* @param idx string, value any
* @return error
**/
func (s *From) Put(idx string, data any) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != s.Host && s.Host != "" {
		return persist.put(s, idx, data)
	}

	key := s.key()
	model, ok := node.models[key]
	if !ok {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.put(idx, data)
}

/**
* Remove: Removes an object from the model
* @param idx string
* @return error
**/
func (s *From) Remove(idx string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != s.Host && s.Host != "" {
		return persist.remove(s, idx)
	}

	key := s.key()
	model, ok := node.models[key]
	if !ok {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.remove(idx)
}

/**
* Get: Gets an object from the model
* @param idx string, dest any
* @return bool, error
**/
func (s *From) Get(idx string, dest any) (bool, error) {
	if !node.started {
		return false, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != s.Host && s.Host != "" {
		return persist.get(s, idx, dest)
	}

	key := s.key()
	model, ok := node.models[key]
	if !ok {
		return false, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.get(idx, dest)
}

/**
* PutObject: Puts an object into the model
* @param idx string, data et.Json
* @return error
**/
func (s *From) PutObject(idx string, data et.Json) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != s.Host && s.Host != "" {
		return persist.putObject(s, idx, data)
	}

	key := s.key()
	model, ok := node.models[key]
	if !ok {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.putObject(idx, data)
}

/**
* GetObject: Gets the model as object
* @param idx string
* @return et.Json, error
**/
func (s *From) GetObject(idx string, dest et.Json) (bool, error) {
	return s.Get(idx, &dest)
}

/**
* RemoveObject: Removes an object from the model
* @param model *Model, key string
* @return error
**/
func (s *From) RemoveObject(idx string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != s.Host && s.Host != "" {
		return persist.removeObject(s, idx)
	}

	key := s.key()
	model, ok := node.models[key]
	if !ok {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.removeObject(key)
}

/**
* IsExisted: Checks if the model exists
* @param field string, idx string
* @return (bool, error)
**/
func (s *From) IsExisted(field, idx string) (bool, error) {
	if !node.started {
		return false, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != s.Host && s.Host != "" {
		return persist.isExisted(s, field, idx)
	}

	key := s.key()
	model, ok := node.models[key]
	if !ok {
		return false, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.isExisted(field, idx)
}

/**
* Count
* @param from *From
* @return (int, error)
**/
func (s *From) Count() (int, error) {
	if !node.started {
		return 0, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != s.Host && s.Host != "" {
		return persist.count(s)
	}

	key := s.key()
	model, ok := node.models[key]
	if !ok {
		return 0, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.count()
}

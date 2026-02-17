package node

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Follow struct{}

/**
* IsExisted: Checks if the object exists
* @param from *From, field, idx string
* @return bool, error
**/
func (s *Follow) IsExisted(from *catalog.From, field, idx string) (bool, error) {
	if node == nil {
		return false, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	key := from.Key()
	node.muModel.RLock()
	model, ok := node.models[key]
	node.muModel.RUnlock()
	if !ok {
		return false, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if !model.IsInit {
		return false, errors.New(msg.MSG_MODEL_NOT_LOAD)
	}

	return model.Get(idx, nil)
}

/**
* RemoveObject
* @param from *From, idx string
* @return error
**/
func (s *Follow) RemoveObject(from *catalog.From, idx string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	key := from.Key()
	node.muModel.RLock()
	model, ok := node.models[key]
	node.muModel.RUnlock()
	if !ok {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if !model.IsInit {
		return errors.New(msg.MSG_MODEL_NOT_LOAD)
	}

	return model.RemoveObject(idx)
}

/**
* putObject
* @param from *From, idx string, data et.Json
* @return error
**/
func (s *Follow) PutObject(from *catalog.From, idx string, data et.Json) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	key := from.Key()
	node.muModel.RLock()
	model, ok := node.models[key]
	node.muModel.RUnlock()
	if !ok {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if !model.IsInit {
		return errors.New(msg.MSG_MODEL_NOT_LOAD)
	}

	return model.PutObject(idx, data)
}

/**
* LoadModel: Loads a model
* @param model *Model
* @return error
**/
func (s *Follow) LoadModel(model *catalog.Model) (*catalog.Model, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	model.IsInit = false
	err := model.Init()
	if err != nil {
		return nil, err
	}

	model.Address = node.Address()
	node.muModel.Lock()
	node.models[model.Key()] = model
	node.muModel.Unlock()

	return model, nil
}

/**
* SetCache: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return error
**/
func (s *Follow) SetCache(key string, value interface{}, duration time.Duration) error {
	_, err := mem.Set(key, value, duration)
	if err != nil {
		return err
	}

	return nil
}

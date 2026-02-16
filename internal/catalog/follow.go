package catalog

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Follow struct{}

/**
* LoadModel: Loads a model
* @param model *Model
* @return error
**/
func (s *Follow) LoadModel(model *Model) (*Model, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	model.IsInit = false
	err := model.Init()
	if err != nil {
		return nil, err
	}

	node.muModel.Lock()
	node.models[model.Key()] = model
	node.muModel.Unlock()

	return model, nil
}

/**
* IsExisted: Checks if the object exists
* @param from *From, field, idx string
* @return bool, error
**/
func (s *Follow) IsExisted(from *From, field, idx string) (bool, error) {
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
func (s *Follow) RemoveObject(from *From, idx string) error {
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
func (s *Follow) PutObject(from *From, idx string, data et.Json) error {
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

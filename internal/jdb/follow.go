package jdb

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
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
* @param key string, value any, now time.Time, duration time.Duration
* @return error
**/
func (s *Follow) SetCache(key string, value any, now time.Time, duration time.Duration) error {
	if !now.IsZero() {
		elapsed := time.Since(now)
		duration -= elapsed
		if duration == 0 {
			return nil
		}
	}

	bt, ok := value.([]byte)
	if !ok {
		var err error
		bt, err = json.Marshal(value)
		if err != nil {
			return err
		}
	}

	node.muCache.Lock()
	node.cache[key] = bt
	node.muCache.Unlock()

	if duration != 0 {
		go func() {
			time.Sleep(duration)
			s.DeleteCache(key)
		}()
		return nil
	}

	return nil
}

/**
* DeleteCache: Deletes a cache value
* @param key string
* @return error
**/
func (s *Follow) DeleteCache(key string) error {
	node.muCache.Lock()
	delete(node.cache, key)
	node.muCache.Unlock()

	return nil
}

/**
* ExistsCache: Deletes a cache value
* @param key string
* @return error
**/
func (s *Follow) ExistsCache(key string) (bool, error) {
	node.muCache.Lock()
	_, ok := node.cache[key]
	node.muCache.Unlock()

	if ok {
		return true, nil
	}

	return false, nil
}

/**
* GetCache: Gets a cache value
* @param key string, dest any
* @return error
**/
func (s *Follow) GetCache(key string, dest any) error {
	node.muCache.Lock()
	bt, ok := node.cache[key]
	node.muCache.Unlock()

	if ok {
		err := json.Unmarshal(bt, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

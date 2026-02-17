package node

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Lead struct{}

/**
* GetDb: Returns a database by name
* @param name string
* @return *catalog.DB, bool
**/
func (s *Lead) GetDb(name string) (*catalog.DB, bool) {
	name = utility.Normalize(name)
	node.muDB.RLock()
	result, ok := node.dbs[name]
	node.muDB.RUnlock()
	if ok {
		return result, true
	}

	err := initDbs()
	if err != nil {
		return nil, false
	}

	exists, err := dbs.Get(name, result)
	if err != nil {
		return nil, false
	}

	if exists {
		node.muDB.Lock()
		node.dbs[name] = result
		node.muDB.Unlock()
		return result, true
	}

	return nil, false
}

/**
* CreateDb: Creates a new database
* @param name string
* @return *catalog.DB, error
**/
func (s *Lead) CreateDb(name string) (*catalog.DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	node.muDB.RLock()
	result, ok := node.dbs[name]
	node.muDB.RUnlock()
	if ok {
		return result, nil
	}

	err := initDbs()
	if err != nil {
		return nil, err
	}

	exists, err := dbs.Get(name, result)
	if err != nil {
		return nil, err
	}

	if !exists {
		result, err = catalog.NewDb(name)
		if err != nil {
			return nil, err
		}

		err = dbs.Put(name, result)
		if err != nil {
			return nil, err
		}
	}

	node.muDB.Lock()
	node.dbs[name] = result
	node.muDB.Unlock()

	return result, nil
}

/**
* DropDb: Drops a database
* @param name string
**/
func (s *Lead) DropDb(name string) error {
	err := initDbs()
	if err != nil {
		return err
	}

	name = utility.Normalize(name)
	err = dbs.Remove(name)
	if err != nil {
		return err
	}

	node.muDB.Lock()
	delete(node.dbs, name)
	node.muDB.Unlock()

	return nil
}

/**
* GetModel: Returns a model by name
* @param from *catalog.From
* @return *catalog.Model, bool
**/
func (s *Lead) GetModel(from *catalog.From) (*catalog.Model, bool) {
	key := from.Key()
	node.muModel.RLock()
	result, ok := node.models[key]
	node.muModel.RUnlock()
	if ok {
		return result, true
	}

	err := initModels()
	if err != nil {
		return nil, false
	}

	exists, err := models.Get(key, result)
	if err != nil {
		return nil, false
	}

	if exists {
		next := node.NextTurn()
		res := node.Request(next, "Follow.LoadModel", from)
		if res.Error != nil {
			return nil, false
		}

		err := res.Get(&result)
		if err != nil {
			return nil, false
		}

		node.muModel.Lock()
		node.models[key] = result
		node.muModel.Unlock()
		return result, true
	}

	return nil, false
}

/**
* DropModel: Drops a model
* @param from *catalog.From
* @return error
**/
func (s *Lead) DropModel(from *catalog.From) error {
	key := from.Key()
	err := initModels()
	if err != nil {
		return err
	}

	key = utility.Normalize(key)
	err = models.Remove(key)
	if err != nil {
		return err
	}

	node.muModel.Lock()
	delete(node.models, key)
	node.muModel.Unlock()

	return nil
}

/**
* SaveModel: Saves a model
* @param model *catalog.Model
* @return error
**/
func (s *Lead) SaveModel(model *catalog.Model) error {
	err := initModels()
	if err != nil {
		return err
	}

	key := model.Key()
	err = models.Put(key, model)
	if err != nil {
		return err
	}

	return nil
}

/**
* SetCache: Sets a cache value
* @param key string, value interface{}, duration time.Duration
* @return error
**/
func (s *Lead) SetCache(key string, value interface{}, now time.Time, duration time.Duration) error {
	if !now.IsZero() {
		elapsed := time.Since(now)
		duration -= elapsed
		if duration == 0 {
			return nil
		}
	}

	node.muCache.Lock()
	node.cache[key] = value
	node.muCache.Unlock()

	if duration != 0 {
		go func() {
			time.Sleep(duration)
			s.DeleteCache(key)
		}()
		return nil
	}

	err := initCache()
	if err != nil {
		return err
	}

	return cache.Put(key, value)
}

/**
* DeleteCache: Deletes a cache value
* @param key string
* @return error
**/
func (s *Lead) DeleteCache(key string) error {
	node.muCache.Lock()
	delete(node.cache, key)
	node.muCache.Unlock()

	err := initCache()
	if err != nil {
		return err
	}

	return cache.Remove(key)
}

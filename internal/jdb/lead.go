package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Lead struct {
	node *Node
}

/**
* GetDb: Returns a database by name
* @param name string
* @return *catalog.DB, bool
**/
func (s *Lead) GetDb(name string) (*catalog.DB, bool) {
	name = utility.Normalize(name)
	s.node.muDB.RLock()
	result, ok := s.node.dbs[name]
	s.node.muDB.RUnlock()
	if ok {
		return result, true
	}

	err := s.node.initDbs()
	if err != nil {
		return nil, false
	}

	exists, err := dbs.Get(name, result)
	if err != nil {
		return nil, false
	}

	if exists {
		s.node.muDB.Lock()
		s.node.dbs[name] = result
		s.node.muDB.Unlock()
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
	s.node.muDB.RLock()
	result, ok := s.node.dbs[name]
	s.node.muDB.RUnlock()
	if ok {
		return result, nil
	}

	err := s.node.initDbs()
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

	s.node.muDB.Lock()
	s.node.dbs[name] = result
	s.node.muDB.Unlock()

	return result, nil
}

/**
* DropDb: Drops a database
* @param name string
**/
func (s *Lead) DropDb(name string) error {
	err := s.node.initDbs()
	if err != nil {
		return err
	}

	name = utility.Normalize(name)
	err = dbs.Remove(name)
	if err != nil {
		return err
	}

	s.node.muDB.Lock()
	delete(s.node.dbs, name)
	s.node.muDB.Unlock()

	return nil
}

/**
* GetModel: Returns a model by name
* @param from *catalog.From
* @return *catalog.Model, bool
**/
func (s *Lead) GetModel(from *catalog.From) (*catalog.Model, bool) {
	key := from.Key()
	s.node.muModel.RLock()
	result, ok := s.node.models[key]
	s.node.muModel.RUnlock()
	if ok {
		return result, true
	}

	err := s.node.initModels()
	if err != nil {
		return nil, false
	}

	exists, err := models.Get(key, result)
	if err != nil {
		return nil, false
	}

	if exists {
		next := s.node.NextTurn()
		res := s.node.Request(next, "Follow.LoadModel", from)
		if res.Error != nil {
			return nil, false
		}

		err := res.Get(&result)
		if err != nil {
			return nil, false
		}

		s.node.muModel.Lock()
		s.node.models[key] = result
		s.node.muModel.Unlock()
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
	err := s.node.initModels()
	if err != nil {
		return err
	}

	key = utility.Normalize(key)
	err = models.Remove(key)
	if err != nil {
		return err
	}

	s.node.muModel.Lock()
	delete(s.node.models, key)
	s.node.muModel.Unlock()

	return nil
}

/**
* SaveModel: Saves a model
* @param model *catalog.Model
* @return error
**/
func (s *Lead) SaveModel(model *catalog.Model) error {
	err := s.node.initModels()
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
* @param key string, value any, duration time.Duration
* @return error
**/
func (s *Lead) SetCache(key string, value any, now time.Time, duration time.Duration) error {
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

	s.node.muCache.Lock()
	s.node.cache[key] = bt
	s.node.muCache.Unlock()

	if duration != 0 {
		go func() {
			time.Sleep(duration)
			s.DeleteCache(key)
		}()
		return nil
	}

	err := s.node.initCache()
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
	s.node.muCache.Lock()
	delete(s.node.cache, key)
	s.node.muCache.Unlock()

	err := s.node.initCache()
	if err != nil {
		return err
	}

	return cache.Remove(key)
}

/**
* ExistsCache: Deletes a cache value
* @param key string
* @return error
**/
func (s *Lead) ExistsCache(key string) (bool, error) {
	s.node.muCache.Lock()
	_, ok := s.node.cache[key]
	s.node.muCache.Unlock()

	if ok {
		return true, nil
	}

	err := s.node.initCache()
	if err != nil {
		return false, err
	}

	exists, err := cache.Exists(key)
	if err != nil {
		return false, err
	}

	return exists, nil
}

/**
* GetCache: Gets a cache value
* @param key string, dest any
* @return error
**/
func (s *Lead) GetCache(key string, dest any) error {
	s.node.muCache.Lock()
	bt, ok := s.node.cache[key]
	s.node.muCache.Unlock()

	if ok {
		err := json.Unmarshal(bt, dest)
		if err != nil {
			return err
		}
		return nil
	}

	err := s.node.initCache()
	if err != nil {
		return err
	}

	_, err = cache.Get(key, dest)
	if err != nil {
		return err
	}

	return nil
}

/**
* CreateSerie: Creates a new serie
* @param tag, format string, value int
* @return error
**/
func (s *Lead) CreateSerie(tag, format string, value int) error {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	err := s.node.initSeries()
	if err != nil {
		return err
	}

	if format == "" {
		format = `%d`
	}

	_, err = Insert(series,
		et.Json{
			"tag":    tag,
			"value":  value,
			"format": format,
		}).
		Execute(nil)

	return err
}

/**
* SetSerie: Sets a serie
* @param tag string, value int
* @return error
**/
func (s *Lead) SetSerie(tag string, value int) error {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	err := s.node.initSeries()
	if err != nil {
		return err
	}

	_, err = Update(series,
		et.Json{
			"value": value,
		}).
		Where(Eq("tag", tag)).
		Execute(nil)

	return err
}

/**
* GetSerie: Gets a serie
* @param tag string
* @return et.Item, error
**/
func (s *Lead) GetSerie(tag string) (et.Item, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return et.Item{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	err := s.node.initSeries()
	if err != nil {
		return et.Item{}, err
	}

	item, err := Update(series,
		et.Json{}).
		BeforeUpdateFn(func(tx *Tx, old, new et.Json) error {
			value := old.Int("value")
			new["value"] = value + 1
			return nil
		}).
		Where(Eq("tag", tag)).
		One(nil)
	if err != nil {
		return et.Item{}, err
	}

	if len(item) == 0 {
		return et.Item{}, errors.New(msg.MSG_INVALID_CONDITION_ONLY_ONE)
	}

	format := item.String("format")
	value := item.Int("value")
	code := fmt.Sprintf(format, value)

	return et.Item{
		Ok: true,
		Result: et.Json{
			"value": value,
			"code":  code,
		}}, nil
}

/**
* DropSerie: Drops a serie
* @param tag string
* @return error
**/
func (s *Lead) DropSerie(tag string) error {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	err := s.node.initSeries()
	if err != nil {
		return err
	}

	_, err = Delete(series).
		Where(Eq("tag", tag)).
		Execute(nil)
	return err
}

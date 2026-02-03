package jql

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/core"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* getDb: Returns a database by name
* @param name string
* @return *DB, error
**/
func getDb(name string) (*dbs.DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	leader, ok := getLeader()
	if ok {
		return syn.getDb(leader, name)
	}

	name = utility.Normalize(name)
	result, ok := s.dbs[name]
	if ok {
		return result, nil
	}

	exists, err := core.GetDb(name, result)
	if err != nil {
		return nil, err
	}

	if exists {
		return result, nil
	}

	if s.isStrict {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
	}

	result, err = dbs.GetDb(name)
	if err != nil {
		return nil, err
	}

	err = core.SetDb(result)
	if err != nil {
		return nil, err
	}

	s.dbs[name] = result

	return result, nil
}

/**
* setDb: Saves the model
* @param db *DB
* @return error
**/
func (s *Node) setDb(db *dbs.DB) error {
	if !s.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	leader, ok := s.getLeader()
	if ok {
		return syn.setDb(leader, db)
	}

	return core.SetDb(db)
}

/**
* getModel
* @param database, schema, name string
* @return *dbs.Model, error
**/
func (s *Node) getModel(database, schema, name string) (*dbs.Model, error) {
	if !s.started {
		return nil, errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(database, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	loadModel := func(result *dbs.Model) (*dbs.Model, error) {
		to := s.nextHost()
		if to == s.Address {
			err := s.loadModel(result)
			if err != nil {
				return nil, err
			}
		} else {
			err := syn.loadModel(to, result)
			if err != nil {
				return nil, err
			}
		}

		return result, nil
	}

	leader, ok := s.getLeader()
	if ok {
		return syn.getModel(leader, database, schema, name)
	}

	key := modelKey(database, schema, name)
	s.modelMu.RLock()
	result, ok := s.models[key]
	s.modelMu.RUnlock()
	if ok {
		return result, nil
	}

	exists, err := core.GetModel(&dbs.From{
		Database: database,
		Schema:   schema,
		Name:     name,
	}, result)
	if err != nil {
		return nil, err
	}

	if exists {
		if result.IsInit {
			return result, nil
		}

		result, err = loadModel(result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	db, err := s.getDb(database)
	if err != nil {
		return nil, err
	}

	if db.IsStrict {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	result, err = db.NewModel(schema, name, false, 1)
	if err != nil {
		return nil, err
	}

	err = core.SetModel(result)
	if err != nil {
		return nil, err
	}

	result, err = loadModel(result)
	if err != nil {
		return nil, err
	}

	s.modelMu.Lock()
	s.models[key] = result
	s.modelMu.Unlock()

	return result, nil
}

/**
* reserveModel
* @param model *Model
* @return error
**/
func (s *Node) loadModel(model *dbs.Model) error {
	if !s.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	err := model.Init()
	if err != nil {
		return err
	}

	key := model.Key()
	s.modelMu.Lock()
	s.models[key] = model
	s.modelMu.Unlock()

	return nil
}

/**
* setModel: Saves the model
* @param model *Model
* @return error
**/
func (s *Node) setModel(model *dbs.Model) error {
	if !s.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}
	if model.IsCore {
		return nil
	}

	leader, ok := s.getLeader()
	if ok {
		return syn.setModel(leader, model)
	}

	return core.SetModel(model)
}

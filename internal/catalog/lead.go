package catalog

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Lead struct{}

/**
* CreateDb: Creates a new database
* @param name string
* @return *DB, error
**/
func (s *Lead) CreateDb(name string) (*DB, error) {
	if node == nil {
		return nil, errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.CreateDb", name)
		if res.Error != nil {
			return nil, res.Error
		}

		var result *DB
		err := res.Get(&result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	node.muDB.Lock()
	result, ok := node.dbs[name]
	node.muDB.Unlock()
	if ok {
		return result, nil
	}

	path := envar.GetStr("DATA_PATH", "./data")
	result = &DB{
		Name:    name,
		Version: node.version,
		Path:    fmt.Sprintf("%s/%s", path, name),
		Schemas: make(map[string]*Schema, 0),
	}
	node.muDB.Lock()
	node.dbs[name] = result
	node.muDB.Unlock()

	return result, nil
}

/**
* GetDb: Returns a database by name
* @param name string
* @return *DB, bool
**/
func (s *Lead) GetDb(name string) (*DB, bool) {
	if node == nil {
		return nil, false
	}

	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.CreateDb", name)
		if res.Error != nil {
			return nil, false
		}

		var result *DB
		var exists bool
		err := res.Get(&result, &exists)
		if err != nil {
			return nil, false
		}

		return result, exists
	}

	name = utility.Normalize(name)
	node.muDB.RLock()
	result, ok := node.dbs[name]
	node.muDB.RUnlock()
	if ok {
		return result, true
	}

	return nil, false
}

/**
* RemoveDb: Removes a database from the global map
* @param name string
**/
func (s *Lead) RemoveDb(name string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.RemoveDb", name)
		if res.Error != nil {
			return res.Error
		}

		return nil
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	delete(node.dbs, name)
	return nil
}

/**
* CoreDb: Returns the core database
* @return *DB, error
**/
func (s *Lead) CoreDb() (*DB, error) {
	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.CoreDb", "")
		if res.Error != nil {
			return nil, res.Error
		}

		return nil, nil
	}

	name := "josefina"
	result, ok := node.dbs[name]
	if ok {
		return result, nil
	}

	return s.CreateDb(name)
}

/**
* GetModel: Returns a model by name
* @param from *From
* @return *Model, bool
**/
func (s *Lead) GetModel(from *From) (*Model, bool) {
	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.GetModel", from)
		if res.Error != nil {
			return nil, false
		}

		var result *Model
		var exists bool
		err := res.Get(&result, &exists)
		if err != nil {
			return nil, false
		}

		return result, exists
	}

	key := from.Key()
	node.muModel.RLock()
	result, ok := node.models[key]
	node.muModel.RUnlock()

	return result, ok
}

/**
* RemoveModel: Drops a model
* @param key string
* @return error
**/
func (s *Lead) RemoveModel(key string) error {
	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.RemoveModel", key)
		if res.Error != nil {
			return res.Error
		}

		return nil
	}

	node.muModel.Lock()
	_, ok := node.models[key]
	if ok {
		delete(node.models, key)
	}
	node.muModel.Unlock()

	return nil
}

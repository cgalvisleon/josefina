package catalog

import (
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
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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

	name = utility.Normalize(name)
	result, ok := node.dbs[name]
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
	node.dbs[name] = result

	return result, nil
}

/**
* GetDb: Returns a database by name
* @param name string
* @return *DB, bool, error
**/
func (s *Lead) GetDb(name string) (*DB, bool, error) {
	if node == nil {
		return nil, false, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.CreateDb", name)
		if res.Error != nil {
			return nil, false, res.Error
		}

		var result *DB
		var exists bool
		err := res.Get(&result, &exists)
		if err != nil {
			return nil, false, err
		}

		return result, exists, nil
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, false, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := node.dbs[name]
	if ok {
		return result, true, nil
	}

	return nil, false, nil
}

/**
* RemoveDb: Removes a database from the global map
* @param name string
**/
func (s *Lead) RemoveDb(name string) error {
	if node == nil {
		return fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
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
	name := "josefina"
	result, ok := node.dbs[name]
	if ok {
		return result, nil
	}

	return s.CreateDb(name)
}

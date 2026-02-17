package node

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	dbs *catalog.Model
)

/**
* initDbs: Initializes the dbs model
* @return error
**/
func initDbs() error {
	if dbs != nil {
		return nil
	}

	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_INITIALIZED)
	}

	db, err := node.coreDb()
	if err != nil {
		return err
	}

	dbs, err = db.NewModel("", "dbs", true, 1)
	if err != nil {
		return err
	}
	if err := dbs.Init(); err != nil {
		return err
	}

	return nil
}

/**
* GetDb: Gets a model
* @param name string, dest *jdb.Model
* @return bool, error
**/
func (s *Node) GetDb(name string) (*catalog.DB, bool) {
	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.GetDb", name)
		if res.Error != nil {
			return nil, false
		}

		var result *catalog.DB
		var exists bool
		err := res.Get(&result, &exists)
		if err != nil {
			return nil, false
		}

		return result, exists
	}

	return nil, false
}

/**
* CreateDb: Creates a new database
* @param name string
* @return *DB, error
**/
func CreateDb(name string) (*catalog.DB, error) {
	err := initDbs()
	if err != nil {
		return nil, err
	}

	var result *catalog.DB
	exists, err := GetDb(name, result)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, errors.New(msg.MSG_DB_NOT_EXISTS)
	}

	result, err = catalog.CreateDb(name)
	if err != nil {
		return nil, err
	}

	key := result.Name
	err = dbs.Put(key, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* DropDb: Removes a db
* @param name string
* @return error
**/
func DropDb(name string) error {
	err := initDbs()
	if err != nil {
		return err
	}

	err = dbs.Remove(name)
	if err != nil {
		return err
	}

	catalog.RemoveDb(name)
	return nil
}

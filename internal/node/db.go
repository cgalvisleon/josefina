package node

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var dbs *catalog.Model

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
* coreDb: Creates the core database
* @return *catalog.DB, error
**/
func (s *Node) coreDb() (*catalog.DB, error) {
	name := "josefina"
	s.muDB.RLock()
	result, ok := s.dbs[name]
	s.muDB.RUnlock()
	if ok {
		return result, nil
	}

	result, err := catalog.NewDb(name)
	if err != nil {
		return nil, err
	}
	s.muDB.Lock()
	s.dbs[name] = result
	s.muDB.Unlock()

	return result, nil
}

/**
* GetDb: Gets a model
* @param name string, dest *jdb.Model
* @return bool, error
**/
func (s *Node) GetDb(name string) (*catalog.DB, bool) {
	leader, imLeader := node.GetLeader()
	if imLeader {
		return s.lead.GetDb(name)
	}

	if leader != nil {
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
func (s *Node) CreateDb(name string) (*catalog.DB, error) {
	leader, imLeader := node.GetLeader()
	if imLeader {
		return s.lead.CreateDb(name)
	}

	if leader != nil {
		res := node.Request(leader, "Leader.CreateDb", name)
		if res.Error != nil {
			return nil, res.Error
		}

		var result *catalog.DB
		err := res.Get(&result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	return nil, errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* DropDb: Removes a db
* @param name string
* @return error
**/
func (s *Node) DropDb(name string) error {
	leader, imLeader := node.GetLeader()
	if imLeader {
		return s.lead.DropDb(name)
	}

	if leader != nil {
		res := node.Request(leader, "Leader.DropDb", name)
		if res.Error != nil {
			return res.Error
		}

		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

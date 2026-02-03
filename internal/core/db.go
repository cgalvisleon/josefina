package core

import (
	"fmt"

	"github.com/cgalvisleon/josefina/internal/mod"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	appName string = "josefina"
	dbs     *mod.Model
)

/**
* initDbs: Initializes the dbs model
* @return error
**/
func initDbs() error {
	if dbs != nil {
		return nil
	}

	db, err := mod.CoreDb()
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
* CreateDb: Creates a new database
* @param name string
* @return *DB, error
**/
func CreateDb(name string) (*mod.DB, error) {
	leader, ok := syn.getLeader()
	if ok {
		return syn.createDb(leader, name)
	}

	err := initDbs()
	if err != nil {
		return nil, err
	}

	var result *mod.DB
	exists, err := GetDb(name, result)
	if err != nil {
		return nil, err
	}

	if exists {
		return result, fmt.Errorf(msg.MSG_DB_EXISTS, name)
	}

	result, err = mod.CreteDb(name)
	if err != nil {
		return nil, err
	}

	bt, err := result.Serialize()
	if err != nil {
		return nil, err
	}

	key := result.Name
	err = dbs.Put(key, bt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* GetDb: Gets a model
* @param name string, dest *jdb.Model
* @return bool, error
**/
func GetDb(name string, dest *mod.DB) (bool, error) {
	leader, ok := syn.getLeader()
	if ok {
		return syn.getDb(leader, name, dest)
	}

	exists, err := mod.GetDb(name, dest)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	}

	err = initDbs()
	if err != nil {
		return false, err
	}

	exists, err = dbs.Get(name, &dest)
	if err != nil {
		return false, err
	}

	if exists {
		mod.AddDb(dest)
	}

	return exists, nil
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

	return dbs.Remove(name)
}

package core

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var (
	appName string = "josefina"
	dbs     *catalog.Model
)

/**
* initDbs: Initializes the dbs model
* @return error
**/
func initDbs() error {
	if dbs != nil {
		return nil
	}

	db, err := catalog.CoreDb()
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
* GetDb: Gets a model
* @param name string, dest *jdb.Model
* @return bool, error
**/
func GetDb(name string, dest *catalog.DB) (bool, error) {
	exists, err := catalog.GetDb(name, dest)
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

	exists, err = dbs.Get(name, dest)
	if err != nil {
		return false, err
	}

	if exists {
		catalog.AddDb(dest)
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

	err = dbs.Remove(name)
	if err != nil {
		return err
	}

	catalog.RemoveDb(name)
	return nil
}

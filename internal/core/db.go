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
* CreteDb: Creates a new database
* @param name string
* @return *DB, error
**/
func CreteDb(name string) (*mod.DB, error) {
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
		return nil, fmt.Errorf(msg.MSG_DB_EXISTS, name)
	}

	bt, err := db.Serialize()
	if err != nil {
		return err
	}

	key := db.Name
	err = dbs.Put(key, bt)
	if err != nil {
		return err
	}

	return nil
}

/**
* GetDb: Gets a model
* @param name string, dest *jdb.Model
* @return bool, error
**/
func GetDb(name string, dest *mod.DB) (bool, error) {
	err := initDbs()
	if err != nil {
		return false, err
	}

	exists, err := dbs.Get(name, &dest)
	if err != nil {
		return false, err
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

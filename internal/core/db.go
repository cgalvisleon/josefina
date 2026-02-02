package core

import (
	"github.com/cgalvisleon/josefina/internal/dbs"
)

var (
	appName string = "josefina"
	mdbs    *dbs.Model
)

/**
* initDbs: Initializes the dbs model
* @return error
**/
func initDbs() error {
	if mdbs != nil {
		return nil
	}

	db, err := dbs.GetDb(appName)
	if err != nil {
		return err
	}

	mdbs, err = db.NewModel("", "dbs", true, 1)
	if err != nil {
		return err
	}
	if err := mdbs.Init(); err != nil {
		return err
	}

	return nil
}

/**
* SetDb: Sets the model
* @param db *DB
* @return error
**/
func SetDb(db *dbs.DB) error {
	err := initDbs()
	if err != nil {
		return err
	}

	bt, err := db.Serialize()
	if err != nil {
		return err
	}

	key := db.Name
	err = mdbs.Put(key, bt)
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
func GetDb(name string, dest *dbs.DB) (bool, error) {
	err := initDbs()
	if err != nil {
		return false, err
	}

	exists, err := mdbs.Get(name, &dest)
	if err != nil {
		return false, err
	}

	return exists, nil
}

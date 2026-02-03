package core

import "github.com/cgalvisleon/josefina/internal/mod"

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

	db, err := mod.GetDb(appName)
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
* SetDb: Sets the model
* @param db *DB
* @return error
**/
func SetDb(db *mod.DB) error {
	err := initDbs()
	if err != nil {
		return err
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

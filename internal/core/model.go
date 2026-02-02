package core

import (
	"github.com/cgalvisleon/josefina/internal/dbs"
)

var (
	models *dbs.Model
)

/**
* initModels: Initializes the models model
* @return error
**/
func initModels() error {
	if models != nil {
		return nil
	}

	db, err := dbs.GetDb(database)
	if err != nil {
		return err
	}

	models, err = db.NewModel("", "models", true, 1)
	if err != nil {
		return err
	}
	if err := models.Init(); err != nil {
		return err
	}

	return nil
}

/**
* SetModel: Sets a model
* @param model *dbs.Model
* @return error
**/
func SetModel(model *dbs.Model) error {
	err := initModels()
	if err != nil {
		return err
	}

	bt, err := model.Serialize()
	if err != nil {
		return err
	}

	key := model.Key()
	err = models.Put(key, bt)
	if err != nil {
		return err
	}

	return nil
}

/**
* getModel: Gets a model
* @param from *dbs.From, dest *dbs.Model
* @return bool, error
**/
func GetModel(from *dbs.From, dest *dbs.Model) (bool, error) {
	err := initModels()
	if err != nil {
		return false, err
	}

	key := from.Key()
	exists, err := models.Get(key, &dest)
	if err != nil {
		return false, err
	}

	return exists, nil
}

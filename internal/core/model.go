package core

import "github.com/cgalvisleon/josefina/internal/mod"

var (
	models *mod.Model
)

/**
* initModels: Initializes the models model
* @return error
**/
func initModels() error {
	if models != nil {
		return nil
	}

	db, err := mod.CoreDb()
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
* @param model *mod.Model
* @return error
**/
func SetModel(model *mod.Model) error {
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
* @param from *mod.From, dest *mod.Model
* @return bool, error
**/
func GetModel(from *mod.From, dest *mod.Model) (bool, error) {
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

/**
* DropModel: Removes a model
* @param name string
* @return error
**/
func DropModel(name string) error {
	err := initModels()
	if err != nil {
		return err
	}

	return models.Remove(name)
}

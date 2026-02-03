package core

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/mod"
	"github.com/cgalvisleon/josefina/internal/msg"
)

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
* CreateModel: Creates a model
* @param database, schema, name string, version int
* @return *mod.Model, error
**/
func CreateModel(database, schema, name string, version int) (*mod.Model, error) {
	leader, ok := syn.getLeader()
	if ok {
		return syn.createModel(leader, database, schema, name, version)
	}

	var result *mod.Model
	exists, err := GetModel(&mod.From{
		Database: database,
		Schema:   schema,
		Name:     name,
	}, result)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, errors.New(msg.MSG_MODEL_NOT_EXISTS)
	}

	var db *mod.DB
	exists, err = GetDb(database, db)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(msg.MSG_DB_NOT_EXISTS)
	}

	result, err = db.NewModel(schema, name, false, version)
	if err != nil {
		return nil, err
	}

	bt, err := result.Serialize()
	if err != nil {
		return nil, err
	}

	key := result.Key()
	err = models.Put(key, bt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* getModel: Gets a model
* @param from *mod.From, dest *mod.Model
* @return bool, error
**/
func GetModel(from *mod.From, dest *mod.Model) (bool, error) {
	leader, ok := syn.getLeader()
	if ok {
		return syn.getModel(leader, from, dest)
	}

	result, exists := mod.GetModel(from)
	if exists {
		*dest = *result
		return true, nil
	}

	err := initModels()
	if err != nil {
		return false, err
	}

	key := from.Key()
	exists, err = models.Get(key, &dest)
	if err != nil {
		return false, err
	}

	return exists, nil
}

/**
* DropModel: Removes a model
* @param from *mod.From
* @return error
**/
func DropModel(from *mod.From) error {
	leader, ok := syn.getLeader()
	if ok {
		return syn.dropModel(leader, from)
	}

	err := initModels()
	if err != nil {
		return err
	}

	key := from.Key()
	err = models.Remove(key)
	if err != nil {
		return err
	}

	return mod.DropModel(key)
}

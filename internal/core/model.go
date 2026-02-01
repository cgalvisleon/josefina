package core

import (
	"errors"

	"github.com/cgalvisleon/josefina/internal/jdb"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var (
	models *jdb.Model
)

/**
* initModels: Initializes the models model
* @return error
**/
func initModels() error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if models != nil {
		return nil
	}

	db, err := node.GetDb(database)
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
* newModel
* @param database, schema, name string, isCore bool, version int
* @return *Model
**/
func newModel(database, schema, name string, isCore bool, version int) (*jdb.Model, error) {
	db, err := node.GetDb(database)
	if err != nil {
		return nil, err
	}
	result, err := db.NewModel(schema, name, isCore, version)
	if err != nil {
		return nil, err
	}

	return result, nil
}

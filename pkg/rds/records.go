package rds

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var records *Model

/**
* initRecords: Initializes the records model
* @return error
**/
func initRecords() error {
	if records != nil {
		return nil
	}

	db, err := newDb(packageName, node.version)
	if err != nil {
		return err
	}

	records, err = db.newModel("", "records", true, 1)
	if err != nil {
		return err
	}
	records.DefineAtrib("schema", TpText, "")
	records.DefineAtrib("model", TpText, "")
	records.DefineAtrib("key", TpText, "")
	records.DefinePrimaryKeys("schema", "model", "key")
	if err := records.init(); err != nil {
		return err
	}

	return nil
}

/**
* setRecord: Sets a record
* @param schema, model, key string
* @return error
**/
func setRecord(schema, model, key string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_FOUND)
	}

	if node.leader != node.host {
		err := methods.setRecord(schema, model, key)
		if err != nil {
			return err
		}

		return nil
	}

	err := initRecords()
	if err != nil {
		return err
	}

	_, err = records.
		Upsert(et.Json{
			"schema": schema,
			"model":  model,
			"key":    key,
		}).
		Execute(nil)
	return err
}

/**
* deleteRecord: Deletes a record
* @param schema, model, key string
* @return error
**/
func deleteRecord(schema, model, key string) error {
	if node == nil {
		return errors.New(msg.MSG_NODE_NOT_FOUND)
	}

	if node.leader != node.host {
		err := methods.setRecord(schema, model, key)
		if err != nil {
			return err
		}

		return nil
	}

	err := initRecords()
	if err != nil {
		return err
	}

	_, err = records.
		Delete().
		Where(Eq("schema", schema)).
		And(Eq("model", model)).
		And(Eq("key", key)).
		Execute(nil)
	return err
}

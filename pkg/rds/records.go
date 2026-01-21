package rds

import "github.com/cgalvisleon/et/et"

var records *Model

/**
* initRecords: Initializes the records model
* @param db *DB
* @return error
**/
func initRecords(db *DB) error {
	var err error
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
	_, err := records.
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
	_, err := records.
		Delete().
		Where(Eq("schema", schema)).
		And(Eq("model", model)).
		And(Eq("key", key)).
		Execute(nil)
	return err
}

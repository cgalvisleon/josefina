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
	_, err := records.upsert(nil, et.Json{
		"schema": schema,
		"model":  model,
		"key":    key,
	})
	return err
}

/**
* deleteRecord: Deletes a record
* @param schema, model, key string
* @return error
**/
func deleteRecord(schema, model, key string) error {
	_, err := records.delete(nil,
		Where(Eq("schema", schema)).
			And(Eq("model", model)).
			And(Eq("key", key)),
	)
	return err
}

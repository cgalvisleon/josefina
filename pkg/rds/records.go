package rds

var records *Model

/**
* initRecords: Initializes the records model
* @return error
**/
func initRecords() error {
	if records != nil {
		return nil
	}

	db, err := getDb(packageName)
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

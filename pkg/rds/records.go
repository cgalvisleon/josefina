package rds

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
	if err := records.init(); err != nil {
		return err
	}

	return nil
}

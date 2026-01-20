package rds

var models *Model

/**
* initModels: Initializes the models model
* @param db *DB
* @return error
**/
func initModels(db *DB) error {
	var err error
	models, err = db.newModel("", "models", true, 1)
	if err != nil {
		return err
	}
	if err := models.init(); err != nil {
		return err
	}

	return nil
}

package josefina

var (
	users      *Model
	series     *Model
	records    *Model
	models     *Model
	references *Model
)

func (s *Tennant) loadCore() error {
	db, err := s.getDb(packageName)
	if err != nil {
		return err
	}

	if err := initUsers(db); err != nil {
		return err
	}

	return nil
}

/**
* initUsers: Initializes the users model
* @param db *DB
* @return error
**/
func initUsers(db *DB) error {
	users, err := db.newModel("", "users", false, 1)
	if err != nil {
		return err
	}

	return nil
}

/**
* initSeries: Initializes the series model
* @param db *DB
* @return error
**/
func initSeries(db *DB) error {
	series, err := db.newModel("", "series", false, 1)
	if err != nil {
		return err
	}
	return nil
}

/**
* initRecords: Initializes the records model
* @param db *DB
* @return error
**/
func initRecords(db *DB) error {
	records, err := db.newModel("", "records", false, 1)
	if err != nil {
		return err
	}
	return nil
}

/**
* initDatabases: Initializes the databases model
* @param db *DB
* @return error
**/
func initDatabases(db *DB) error {
	databases, err := db.newModel("", "databases", false, 1)
	if err != nil {
		return err
	}
	return nil
}

/**
* initSchemas: Initializes the schemas model
* @param db *DB
* @return error
**/
func initSchemas(db *DB) error {
	schemas, err := db.newModel("", "schemas", false, 1)
	if err != nil {
		return err
	}
	return nil
}

/**
* initModels: Initializes the models model
* @param db *DB
* @return error
**/
func initModels(db *DB) error {
	models, err := db.newModel("", "models", false, 1)
	if err != nil {
		return err
	}
	return nil
}

/**
* initReferences: Initializes the references model
* @param db *DB
* @return error
**/
func initReferences(db *DB) error {
	references, err := db.newModel("", "references", false, 1)
	if err != nil {
		return err
	}
	return nil
}

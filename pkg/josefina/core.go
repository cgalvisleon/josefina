package josefina

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

var (
	users      *Model
	series     *Model
	records    *Model
	databases  *Model
	schemas    *Model
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
	if err := initSeries(db); err != nil {
		return err
	}
	if err := initRecords(db); err != nil {
		return err
	}
	if err := initDatabases(db); err != nil {
		return err
	}
	if err := initSchemas(db); err != nil {
		return err
	}
	if err := initModels(db); err != nil {
		return err
	}
	if err := initReferences(db); err != nil {
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
	var err error
	users, err = db.newModel("", "users", true, 1)
	if err != nil {
		return err
	}
	if err := users.init(); err != nil {
		return err
	}

	if users.count() == 0 {
		useranme := envar.GetStr("USERNAME", "admin")
		password := envar.GetStr("PASSWORD", "admin")
		users.insert(et.Json{
			"username": useranme,
			"password": password,
		})
	}

	return nil
}

/**
* initSeries: Initializes the series model
* @param db *DB
* @return error
**/
func initSeries(db *DB) error {
	var err error
	series, err = db.newModel("", "series", true, 1)
	if err != nil {
		return err
	}
	if err := series.init(); err != nil {
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

/**
* initDatabases: Initializes the databases model
* @param db *DB
* @return error
**/
func initDatabases(db *DB) error {
	var err error
	databases, err = db.newModel("", "databases", true, 1)
	if err != nil {
		return err
	}
	if err := databases.init(); err != nil {
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
	var err error
	schemas, err = db.newModel("", "schemas", true, 1)
	if err != nil {
		return err
	}
	if err := schemas.init(); err != nil {
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

/**
* initReferences: Initializes the references model
* @param db *DB
* @return error
**/
func initReferences(db *DB) error {
	var err error
	references, err = db.newModel("", "references", true, 1)
	if err != nil {
		return err
	}
	if err := references.init(); err != nil {
		return err
	}

	return nil
}

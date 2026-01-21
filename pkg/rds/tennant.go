package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

const (
	packageName = "josefina"
)

type Tennant struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Path    string         `json:"path"`
	Dbs     map[string]*DB `json:"dbs"`
}

/**
* loadTennant
* @param name string
* @return *Tennant, error
**/
func loadTennant(path, name, version string) (*Tennant, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	result := &Tennant{
		Name:    name,
		Version: version,
		Path:    path,
		Dbs:     make(map[string]*DB),
	}
	err := result.loadCore()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* loadCore
* @return error
**/
func (s *Tennant) loadCore() error {
	db, err := s.newDb(packageName)
	if err != nil {
		return err
	}
	if err := initTransactions(db); err != nil {
		return err
	}
	if err := initDatabases(db); err != nil {
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
	if err := initModels(db); err != nil {
		return err
	}
	if err := db.save(); err != nil {
		return err
	}
	if err := s.loadDbs(); err != nil {
		return err
	}

	return nil
}

/**
* newDb
* @param name string
* @return *DB, error
**/
func (s *Tennant) newDb(name string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := s.Dbs[name]
	if ok {
		return result, nil
	}

	result = &DB{
		Name:    name,
		Version: s.Version,
		Path:    fmt.Sprintf("%s/%s", s.Path, name),
		Schemas: make(map[string]*Schema),
		tennant: s,
	}
	s.Dbs[name] = result

	return result, nil
}

/**
* loadDb
* @param name string
* @return *DB, error
**/
func (s *Tennant) getDb(name string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := s.Dbs[name]
	if ok {
		return result, nil
	}

	return nil, fmt.Errorf(msg.MSG_DB_NOT_FOUND, name)
}

/**
* loadDbs
* @return error
**/
func (s *Tennant) loadDbs() error {
	if databases == nil {
		return fmt.Errorf(msg.MSG_DONT_HAVE_DATABASES)
	}

	databases.Where()

	return nil
}

/**
* getModel
* @param database string, schema string, model string
* @return *Model, error
**/
func (s *Tennant) getModel(database, schema, name string) (*Model, error) {
	db, err := s.getDb(database)
	if err != nil {
		return nil, err
	}

	return db.getModel(schema, name)
}

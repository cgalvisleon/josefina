package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Node struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Path    string         `json:"path"`
	Dbs     map[string]*DB `json:"dbs"`
}

/**
* newDb
* @param name string
* @return *DB, error
**/
func (s *Node) newDb(name string) (*DB, error) {
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
	}
	s.Dbs[name] = result

	return result, nil
}

/**
* load
* @return error
**/
func (s *Node) load() error {
	db, err := s.newDb(packageName)
	if err != nil {
		return err
	}
	db.isCore = true

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

	return nil
}

/**
* getDb
* @param name string
* @return *DB, error
**/
func (s *Node) getDb(name string) (*DB, error) {
	if db == nil {
		return fmt.Errorf(msg.MSG_DB_NOT_FOUND)
	}

	data, err := db.toJson()
	if err != nil {
		return err
	}

	logs.Debug(data.ToString())
	err = db.load(s)
	if err != nil {
		return err
	}

	s.Dbs[db.Name] = db
	return nil
}

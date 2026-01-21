package rds

import (
	"fmt"
	"os"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Node struct {
	Host    string            `json:"name"`
	Version string            `json:"version"`
	Path    string            `json:"path"`
	Dbs     map[string]*DB    `json:"dbs"`
	db      *DB               `json:"-"`
	models  map[string]*Model `json:"-"`
}

/**
* newNode
* @param version, path string
* @return *Node, error
**/
func newNode(version, path string) (*Node, error) {
	hostName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &Node{
		Host:    hostName,
		Version: version,
		Path:    path,
		Dbs:     make(map[string]*DB),
		db:      newDb(path, packageName, version),
	}, nil
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

	result = newDb(s.Path, name, s.Version)
	s.Dbs[name] = result

	return result, nil
}

/**
* load
* @return error
**/
func (s *Node) load() error {
	if err := initTransactions(s.db); err != nil {
		return err
	}
	if err := initDatabases(s.db); err != nil {
		return err
	}
	if err := initUsers(s.db); err != nil {
		return err
	}
	if err := initSeries(s.db); err != nil {
		return err
	}
	if err := initRecords(s.db); err != nil {
		return err
	}
	if err := initModels(s.db); err != nil {
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

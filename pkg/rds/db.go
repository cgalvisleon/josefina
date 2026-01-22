package rds

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var databases *Model

/**
* initDatabases: Initializes the databases model
* @param db *DB
* @return error
**/
func initDatabases() error {
	if databases != nil {
		return nil
	}

	db, err := newDb(packageName, node.version)
	if err != nil {
		return err
	}

	databases, err = db.newModel("", "databases", true, 1)
	if err != nil {
		return err
	}
	if err := databases.init(); err != nil {
		return err
	}

	return nil
}

type DB struct {
	Name    string             `json:"name"`
	Version string             `json:"version"`
	Path    string             `json:"path"`
	Schemas map[string]*Schema `json:"schemas"`
}

/**
* newDb: Creates a new database
* @param name string, version string
* @return *DB, error
**/
func newDb(name, version string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := node.dbs[name]
	if ok {
		return result, nil
	}

	path := envar.GetStr("PATH_DATA", "./data")
	result = &DB{
		Name:    name,
		Version: version,
		Path:    fmt.Sprintf("%s/%s", path, name),
		Schemas: make(map[string]*Schema, 0),
	}
	node.dbs[name] = result

	return result, nil
}

/**
* serialize
* @return []byte, error
**/
func (s *DB) serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* toJson
* @return et.Json, error
**/
func (s *DB) toJson() (et.Json, error) {
	definition, err := s.serialize()
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(definition, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* save: Saves the database
* @return error
**/
func (s *DB) save() error {
	if databases == nil {
		return nil
	}

	if !databases.isInit {
		return nil
	}

	scr, err := s.serialize()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s", s.Name)
	err = databases.put(key, scr)
	if err != nil {
		return err
	}

	return nil
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema
**/
func (s *DB) getSchema(name string) *Schema {
	name = utility.Normalize(name)
	result, ok := s.Schemas[name]
	if !ok {
		result = &Schema{
			Database: s.Name,
			Name:     name,
			Models:   make(map[string]*Model, 0),
			db:       s,
		}
	}

	return result
}

/**
* getModel: Returns a model by schema and name
* @param schema, name, host string
* @return *Model, error
**/
func (s *DB) getModel(schema, name, host string) (*Model, error) {
	sch := s.getSchema(schema)
	return sch.getModel(name, host)
}

/**
* newModel: Creates a new model
* @param schema string, name string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) newModel(schema, name string, isCore bool, version int) (*Model, error) {
	sch := s.getSchema(schema)
	return sch.newModel(name, isCore, version)
}

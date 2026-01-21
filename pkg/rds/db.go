package rds

import (
	"encoding/json"
	"errors"
	"fmt"

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
func initDatabases(db *DB) error {
	if databases != nil {
		return nil
	}

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

type DB struct {
	Name    string             `json:"name"`
	Version string             `json:"version"`
	Path    string             `json:"path"`
	Schemas map[string]*Schema `json:"schemas"`
}

/**
* newDb
* @param path, name, version string
* @return *DB
**/
func newDb(path, name, version string) *DB {
	return &DB{
		Name:    name,
		Version: version,
		Path:    fmt.Sprintf("%s/%s", path, name),
		Schemas: make(map[string]*Schema, 0),
	}
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
func (s *DB) getSchema(name string) (*Schema, error) {
	result, ok := s.Schemas[name]
	if !ok {
		return nil, errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	return result, nil
}

/**
* getModel: Returns a model by schema and name
* @param schema string, name string
* @return *Model, error
**/
func (s *DB) getModel(schema, name string) (*Model, error) {
	sch, err := s.getSchema(schema)
	if err != nil {
		return nil, err
	}

	return sch.getModel(name)
}

/**
* newSchema: Creates a new schema
* @param name string
* @return *Schema
**/
func (s *DB) newSchema(name string) *Schema {
	name = utility.Normalize(name)
	result, ok := s.Schemas[name]
	if ok {
		return result
	}

	result = &Schema{
		Database: s.Name,
		Name:     name,
		Models:   make(map[string]*Model, 0),
		db:       s,
	}

	s.Schemas[name] = result
	return result
}

/**
* newModel: Creates a new model
* @param schema string, name string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) newModel(schema, name string, isCore bool, version int) (*Model, error) {
	sch := s.newSchema(schema)
	return sch.newModel(name, isCore, version)
}

package jdb

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var (
	version string = "0.0.1"
)

type DB struct {
	Name     string             `json:"name"`
	Version  string             `json:"version"`
	Path     string             `json:"path"`
	Schemas  map[string]*Schema `json:"schemas"`
	IsStrict bool               `json:"is_strict"`
	isDebug  bool               `json:"-"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *DB) Serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *DB) ToJson() (et.Json, error) {
	definition, err := s.Serialize()
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
* SetDebug
* @param debug bool
**/
func (s *DB) SetDebug(debug bool) {
	s.isDebug = debug
	for _, schema := range s.Schemas {
		for _, model := range schema.Models {
			model.isDebug = debug
		}
	}
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema
**/
func (s *DB) getSchema(name string) *Schema {
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
* NewModel: Creates a new model
* @param schema, name	string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) NewModel(schema, name string, isCore bool, version int) (*Model, error) {
	sch := s.getSchema(schema)
	model, err := sch.newModel(name, isCore, version)
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* GetDb: Returns a database by name
* @param name string
* @return *DB, error
**/
func GetDb(name string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	path := envar.GetStr("DATA_PATH", "./data")
	result := &DB{
		Name:    name,
		Version: version,
		Path:    fmt.Sprintf("%s/%s", path, name),
		Schemas: make(map[string]*Schema, 0),
	}

	return result, nil
}

package jdb

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

type OnSave func(key string, data any) error

type DB struct {
	Name     string             `json:"name"`
	Version  string             `json:"version"`
	Path     string             `json:"path"`
	Schemas  map[string]*Schema `json:"schemas"`
	IsStrict bool               `json:"is_strict"`
	isDebug  bool               `json:"-"`
	onSave   OnSave             `json:"-"`
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
* ToJson
* @return et.Json, error
**/
func (s *DB) ToJson() (et.Json, error) {
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
* SetOnSave
* @param onSave OnSave
**/
func (s *DB) SetOnSave(onSave OnSave) {
	s.onSave = onSave
}

/**
* Save
* @return error
**/
func (s *DB) Save() error {
	if s.onSave == nil {
		return nil
	}

	return s.onSave(s.Name, s)
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

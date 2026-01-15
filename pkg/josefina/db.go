package josefina

import (
	"errors"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type DB struct {
	Name    string             `json:"name"`
	Version string             `json:"version"`
	Path    string             `json:"path"`
	Schemas map[string]*Schema `json:"schemas"`
	Models  map[string]*Model  `json:"models"`
	tennant *Tennant           `json:"-"`
}

func (s *DB) save() error {
	if s.tennant == nil {
		return errors.New(msg.MSG_TENNANT_NOT_FOUND)
	}

	return s.tennant.save()
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema
**/
func (s *DB) getSchema(name string) *Schema {
	return s.newSchema(name)
}

/**
* getModel: Returns a model by schema and name
* @param schema string, name string
* @return *Model, error
**/
func (s *DB) getModel(schema, name string) (*Model, error) {
	sch := s.getSchema(schema)
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

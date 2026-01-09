package josefina

import (
	"fmt"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type DB struct {
	Name    string             `json:"name"`
	Version string             `json:"version"`
	Path    string             `json:"path"`
	Schemas map[string]*Schema `json:"schemas"`
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema, error
**/
func (s *DB) getSchema(name string) (*Schema, error) {
	return s.newSchema(name)
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
* @return *Schema, error
**/
func (s *DB) newSchema(name string) (*Schema, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := s.Schemas[name]
	if ok {
		return result, nil
	}

	result = &Schema{
		Database: s.Name,
		Name:     name,
		Models:   make(map[string]*Model, 0),
	}

	s.Schemas[name] = result
	return result, nil
}

/**
* newModel: Creates a new model
* @param schema string, name string
* @return *Model, error
**/
func (s *DB) newModel(schema, name string, version int) (*Model, error) {
	sch, err := s.newSchema(schema)
	if err != nil {
		return nil, err
	}

	return sch.newModel(name, version)
}

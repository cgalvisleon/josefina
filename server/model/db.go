package model

import (
	"errors"

	"github.com/cgalvisleon/josefina/server/msg"
)

var dbs DBS

func init() {
	dbs = make(DBS)
}

type DBS map[string]*DB

type DB struct {
	Name    string             `json:"name"`
	Version int                `json:"version"`
	Release int                `json:"release"`
	Schemas map[string]*Schema `json:"schemas"`
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema, error
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

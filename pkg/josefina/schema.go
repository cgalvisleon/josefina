package josefina

import (
	"errors"

	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Schema struct {
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
	db       *DB               `json:"-"`
}

/**
* newModel: Returns a new model
* @param name string, version int
* @return *Model
**/
func (s *Schema) newModel(name string, version int) (*Model, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	result, ok := s.Models[name]
	if ok {
		return result, nil
	}

	name = utility.Normalize(name)
	result = &Model{
		From: &From{
			Database: s.Database,
			Schema:   s.Name,
			Name:     name,
		},
		Indexes:       make([]string, 0),
		PrimaryKeys:   make([]string, 0),
		Unique:        make([]string, 0),
		Required:      make([]string, 0),
		Hidden:        make([]string, 0),
		References:    make([]string, 0),
		Master:        make(map[string]*Master, 0),
		Details:       make(map[string]*Detail, 0),
		Rollups:       make(map[string]*Detail, 0),
		Relations:     make(map[string]*Detail, 0),
		BeforeInserts: make([]*Trigger, 0),
		BeforeUpdates: make([]*Trigger, 0),
		BeforeDeletes: make([]*Trigger, 0),
		AfterInserts:  make([]*Trigger, 0),
		AfterUpdates:  make([]*Trigger, 0),
		AfterDeletes:  make([]*Trigger, 0),
		Version:       version,
		db:            s.db,
	}

	s.Models[name] = result
	name = strs.Append(s.Name, name, ".")
	s.db.Models[name] = result
	return result, nil
}

/**
* getModel: Returns a model
* @param name string
* @return *Model
**/
func (s *Schema) getModel(name string) (*Model, error) {
	return s.newModel(name, 1)
}

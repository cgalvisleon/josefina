package db

import (
	"errors"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/server/msg"
)

type Schema struct {
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
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
	}

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

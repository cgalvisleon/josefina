package rds

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/cgalvisleon/josefina/pkg/store"
)

type Schema struct {
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
	db       *DB               `json:"-"`
}

/**
* newModel: Returns a new model
* @param name string, isCore bool, version int
* @return *Model
**/
func (s *Schema) newModel(name string, isCore bool, version int) (*Model, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	result, ok := s.Models[name]
	if ok {
		return result, nil
	}

	name = utility.Normalize(name)
	host := fmt.Sprintf(`%s:%d`, node.Host, node.Port)
	result = &Model{
		From: &From{
			Database: s.Database,
			Schema:   s.Name,
			Name:     name,
			Host:     host,
			Fields:   make(map[string]*Field, 0),
		},
		Indexes:       make([]string, 0),
		PrimaryKeys:   make([]string, 0),
		Unique:        make([]string, 0),
		Required:      make([]string, 0),
		Hidden:        make([]string, 0),
		References:    make(map[string]*Detail, 0),
		Details:       make(map[string]*Detail, 0),
		Rollups:       make(map[string]*Detail, 0),
		Relations:     make(map[string]*Detail, 0),
		Calcs:         make(map[string][]byte, 0),
		BeforeInserts: make([]*Trigger, 0),
		BeforeUpdates: make([]*Trigger, 0),
		BeforeDeletes: make([]*Trigger, 0),
		AfterInserts:  make([]*Trigger, 0),
		AfterUpdates:  make([]*Trigger, 0),
		AfterDeletes:  make([]*Trigger, 0),
		Version:       version,
		IsCore:        isCore,
		db:            s.db,
		data:          make(map[string]*store.FileStore, 0),
		triggers:      make(map[string]*Vm, 0),
	}
	_, err := result.defineIndexField()
	if err != nil {
		return nil, err
	}
	s.Models[name] = result
	return result, nil
}

/**
* getModel: Returns a model
* @param name string
* @return *Model
**/
func (s *Schema) getModel(name string) (*Model, error) {
	result, ok := s.Models[name]
	if !ok {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return result, nil
}

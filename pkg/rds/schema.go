package rds

import (
	"errors"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Schema struct {
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
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

	result, err := newModel(s.Database, s.Name, name, isCore, version)
	if err != nil {
		return nil, err
	}
	s.Models[name] = result

	return result, nil
}

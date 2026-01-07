package model

import (
	"errors"

	"github.com/cgalvisleon/josefina/server/msg"
)

type Schema struct {
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
}

/**
* getModel: Returns a model by name
* @param name string
* @return *Model, error
**/
func (s *Schema) getModel(name string) (*Model, error) {
	result, ok := s.Models[name]
	if !ok {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return result, nil
}

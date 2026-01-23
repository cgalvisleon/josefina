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

	s.Models[name] = result
	s.db.Schemas[s.Name] = s
	return result, nil
}

/**
* getModel: Returns a model
* @param name, host string, model *Model
* @return bool, error
**/
func (s *Schema) getModel(name, host string, model *Model) (bool, error) {
	name = utility.Normalize(name)
	result, ok := s.Models[name]
	if ok {
		model = result
		return true, nil
	}

	err := initModels()
	if err != nil {
		return false, err
	}

	key := modelKey(s.Database, s.Name, name)
	exists, err := models.get(key, &result)
	if err != nil {
		return nil, err
	}

	if exists {
		result.host = host
		s.Models[name] = result
		return result, nil
	}

	return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
}

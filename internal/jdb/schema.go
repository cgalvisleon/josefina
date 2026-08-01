package jdb

import (
	"errors"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

/**
* Schema: Represents a schema in the database
**/
type Schema struct {
	Database string            `json:"database"` // Database name
	Name     string            `json:"name"`     // Schema name
	models   map[string]*Model `json:"-"`        // Models
	db       *DB               `json:"-"`        // Database
	mu       *sync.RWMutex     `json:"-"`        // Mutex
}

func (s *DB) newSchema(name string) *Schema {
	return &Schema{
		Database: s.Name,
		Name:     name,
		models:   make(map[string]*Model),
		db:       s,
		mu:       &sync.RWMutex{},
	}
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Schema) ToJson() et.Json {
	models := et.Json{}
	for _, model := range s.models {
		models[model.Name] = et.Json{
			"name":    model.Name,
			"path":    model.Path,
			"version": model.Version,
			"is_core": model.IsCore,
		}
	}

	return et.Json{
		"database": s.Database,
		"name":     s.Name,
		"models":   models,
	}
}

/**
* addModel: Adds a model to the schema
* @param model *Model
**/
func (s *Schema) addModel(model *Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[model.Name] = model
}

/**
* getModel: Returns a model from the schema
* @param name string
* @return *Model, bool
**/
func (s *Schema) getModel(name string) (*Model, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	model, exists := s.models[name]
	return model, exists
}

/**
* removeModel: Removes a model from the schema
* @param name string
* @return bool
**/
func (s *Schema) removeModel(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.models[name]
	if exists {
		delete(s.models, name)
	}
	return exists
}

/**
* DeleteModel: Deletes a model
* @param string name
* @return error
 */
func (s *Schema) DeleteModel(name string) error {
	model, exists := s.getModel(name)
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if err := model.Empty(); err != nil {
		return err
	}

	s.removeModel(name)
	return nil
}

/**
* GetModel: Returns a model
* @param string name
* @return *Model, error
**/
func (s *Schema) GetModel(name string) (*Model, error) {
	name = store.Normalize(name)
	result, exists := s.getModel(name)
	if !exists {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return result, nil
}

/**
* Empty: Empties the schema
* @return error
**/
func (s *Schema) Empty() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, model := range s.models {
		err := model.Empty()
		if err != nil {
			return err
		}
	}

	s.models = make(map[string]*Model, 0)
	return nil
}

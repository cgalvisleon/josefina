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
	Models   map[string]*Model `json:"models"`   // Models
	db       *DB               `json:"-"`        // Database
	mu       *sync.RWMutex     `json:"-"`        // Mutex
}

func newSchema(db *DB, name string) *Schema {
	return &Schema{
		Database: db.Name,
		Name:     name,
		Models:   make(map[string]*Model),
		db:       db,
		mu:       &sync.RWMutex{},
	}
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Schema) ToJson() et.Json {
	models := et.Json{}
	for _, model := range s.Models {
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
* save: Save schema data
* @return error
**/
func (s *Schema) save() error {
	if s.db == nil {
		return errors.New(msg.MSG_DB_IS_NIL)
	}

	if s.db.schemas == nil {
		return errors.New(msg.MSG_SCHEMAS_IS_NIL)
	}

	err := s.db.schemas.putObject(s.Name, s.ToJson())
	if err != nil {
		return err
	}

	return nil
}

/**
* addModel: Adds a model to the schema
* @param model *Model
**/
func (s *Schema) addModel(model *Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Models[model.Name] = model
}

/**
* getModel: Returns a model from the schema
* @param name string
* @return *Model, bool
**/
func (s *Schema) getModel(name string) (*Model, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	model, exists := s.Models[name]
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
	_, exists := s.Models[name]
	if exists {
		delete(s.Models, name)
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

	return s.save()
}

/**
* GetModel: Returns a model
* @param string name
* @return *Model, error
**/
func (s *Schema) GetModel(name string) (*Model, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = store.Normalize(name)
	result, exists := s.models[name]
	if !exists {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return result, nil
}

/**
* ListModels: Returns all models in this schema.
* @return []*Model
**/
func (s *Schema) ListModels() []*Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Model, 0, len(s.models))
	for _, m := range s.models {
		result = append(result, m)
	}
	return result
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

	return s.save()
}

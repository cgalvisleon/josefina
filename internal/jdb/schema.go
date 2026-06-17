package jdb

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

/**
* loadSchemas: Loads the schemas
* @return error
**/
func (s *DB) loadSchemas() error {
	model, err := s.Define(DModel{
		Schema:  sysSchema,
		Name:    "schemas",
		IsCore:  true,
		Version: 1,
	})
	if err != nil {
		return err
	}

	err = model.Init()
	if err != nil {
		return err
	}

	s.schemas = model
	cursor, err := s.schemas.NewCursor(false, 0, 0)
	if err != nil {
		return err
	}

	defer cursor.Close()
	for cursor.Next() {
		var schema *Schema
		err := cursor.Scan(&schema)
		if err != nil {
			return err
		}

		err = schema.load(s)
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

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

/**
* load: Loads the transaction
* @param db *DB
* @return error
**/
func (s *Schema) load(db *DB) error {
	s.db = db
	s.mu = &sync.RWMutex{}

	return nil
}

/**
* save: Save schema data
* @return error
**/
func (s *Schema) save() error {
	if s.db == nil {
		return errors.New(msg.MSG_DB_IS_NIL)
	}

	err := s.db.schemas.Put(s.Name, s)
	if err != nil {
		return err
	}

	return nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Schema) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
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
* newModel: Returns a new model
* @param name string, isCore bool, version int
* @return *Model
**/
func (s *Schema) newModel(name string, isCore bool, version int) (*Model, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	name = store.Normalize(name)
	result, exists := s.getModel(name)
	if exists {
		return result, nil
	}

	path := filepath.Join(s.db.Path, s.Name)
	path = filepath.Join(path, name)
	result, err := newModel(s, name, path, version, isCore)
	if err != nil {
		return nil, err
	}

	s.addModel(result)

	return result, nil
}

/**
* DeleteModel: Deletes a model
* @param string name
* @return error
 */
func (s *Schema) DeleteModel(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = store.Normalize(name)

	model, exists := s.Models[name]
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if err := model.Empty(); err != nil {
		return err
	}

	delete(s.models, name)

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

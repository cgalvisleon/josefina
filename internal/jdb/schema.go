package jdb

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* loadSchemas: Loads the schemas
* @param db *DB
* @return error
**/
func loadSchemas(db *DB) error {
	result, err := db.NewModel("", "schemas", true, 1)
	if err != nil {
		return err
	}

	err = result.Init()
	if err != nil {
		return err
	}

	db.schemas = result
	db.schemas.ForEachBt(func(idx string, src []byte) (bool, error) {
		var schema Schema
		if err := json.Unmarshal(src, &schema); err != nil {
			return false, err
		}

		err := schema.load(db)
		if err != nil {
			return false, err
		}
		return true, nil
	}, false, 0, 0)
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
	for _, model := range s.Models {
		err := model.load(s)
		if err != nil {
			return err
		}
	}

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

	err := s.db.transaction.Put(s.Name, s)
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
* NewModel: Returns a new model
* @param name string, isCore bool, version int
* @return *Model
**/
func (s *Schema) NewModel(name string, isCore bool, version int) (*Model, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	name = utility.Normalize(name)

	s.mu.RLock()
	result, exists := s.Models[name]
	s.mu.RUnlock()
	if exists {
		return result, nil
	}

	path := filepath.Join(s.db.Path, s.Name)
	path = filepath.Join(path, name)
	result, err := newModel(s, name, path, version, isCore)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Models[result.Name] = result

	err = s.save()
	if err != nil {
		return nil, err
	}

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

	name = utility.Normalize(name)

	model, exists := s.Models[name]
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	if err := model.Empty(); err != nil {
		return err
	}

	delete(s.Models, name)

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

	name = utility.Normalize(name)
	result, exists := s.Models[name]
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

	for _, model := range s.Models {
		err := model.Empty()
		if err != nil {
			return err
		}
	}

	s.Models = make(map[string]*Model, 0)

	return s.save()
}

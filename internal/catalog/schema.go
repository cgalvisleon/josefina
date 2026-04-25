package catalog

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

type Schema struct {
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
	db       *DB               `json:"-"`
	mu       sync.RWMutex      `json:"-"`
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

	path := strs.Append(s.db.Path, s.Name, "/")
	path = fmt.Sprintf("%s/%s", path, name)
	result = &Model{
		Database:      s.Database,
		Schema:        s.Name,
		Name:          name,
		Path:          path,
		Fields:        make(map[string]*Field, 0),
		Indexes:       make([]string, 0),
		PrimaryKeys:   make([]string, 0),
		ForeignKeys:   make(map[string]*Detail, 0),
		Unique:        make([]string, 0),
		Required:      make([]string, 0),
		Hidden:        make([]string, 0),
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
		stores:        make(map[string]*store.FileStore, 0),
		schema:        s,
	}
	_, err := result.defineIndexField()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Models[result.Name] = result

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
	model, exists := s.Models[name]
	if !exists {
		return errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	err := model.Empty()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Models, name)

	return nil
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

	return nil
}

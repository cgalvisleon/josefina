package catalog

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* Schema: Represents a schema in the database
**/
type Schema struct {
	Database string            `json:"database"` // Database name
	Name     string            `json:"name"`     // Schema name
	Models   map[string]*Model `json:"models"`   // Models
	db       *DB               `json:"-"`        // Database
	mu       sync.RWMutex      `json:"-"`        // Mutex
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

	return nil
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

	return nil
}

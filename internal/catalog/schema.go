package catalog

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

type Schema struct {
	Database string           `json:"database"`
	Name     string           `json:"name"`
	Models   map[string]*From `json:"models"`
	db       *DB              `json:"-"`
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

	name = utility.Normalize(name)
	path := strs.Append(s.db.Path, s.Name, "/")
	path = fmt.Sprintf("%s/%s", path, name)
	result := &Model{
		From: &From{
			Database: s.Database,
			Schema:   s.Name,
			Name:     name,
		},
		Fields:        make(map[string]*Field, 0),
		Path:          path,
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
		Triggers:      make([]*Trigger, 0),
		beforeInserts: make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
		Version:       version,
		IsCore:        isCore,
		stores:        make(map[string]*store.FileStore, 0),
		schema:        s,
	}
	_, err := result.defineIndexField()
	if err != nil {
		return nil, err
	}
	s.Models[name] = result.From

	return result, nil
}

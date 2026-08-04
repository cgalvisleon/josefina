package jdb

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
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

/**
* newModel: Creates a new model
* @param string name, string path, int version, bool isCore
* @return (*Model, error)
**/
func (s *Schema) newModel(name string, version int, isCore bool) (*Model, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, errors.New(msg.MSG_NAME_IS_REQUIRED)
	}

	name = store.Normalize(name)
	result := &Model{
		Database:      s.Database,
		Schema:        s.Name,
		Name:          name,
		Path:          s.db.Path,
		Fields:        make(map[string]*Field, 0),
		Indexes:       make([]*Index, 0),
		PrimaryKeys:   make([]string, 0),
		ForeignKeys:   make(map[string]*Detail, 0),
		Unique:        make([]*Index, 0),
		Required:      make([]*Index, 0),
		Hidden:        make([]string, 0),
		Details:       make(map[string]*Detail, 0),
		Masters:       make(map[string]*Detail, 0),
		Rollups:       make(map[string]*Detail, 0),
		Relations:     make(map[string]*Detail, 0),
		Calcs:         make(map[string]string, 0),
		BeforeInserts: make([]*Trigger, 0),
		BeforeUpdates: make([]*Trigger, 0),
		BeforeDeletes: make([]*Trigger, 0),
		AfterInserts:  make([]*Trigger, 0),
		AfterUpdates:  make([]*Trigger, 0),
		AfterDeletes:  make([]*Trigger, 0),
		Mode:          store.ReadWrite,
		Version:       version,
		IsCore:        isCore,
		IsChangue:     false,
		schema:        s,
		db:            s.db,
		mu:            map[string]*sync.RWMutex{},
		stores:        make(map[string]*store.FileStore, 0),
		btrees:        make(map[string]*BTree, 0),
		onPut:         make([]TriggerFnBt, 0),
		onRemove:      make([]TriggerFnBt, 0),
	}
	_, err := result.defineSource()
	if err != nil {
		return nil, err
	}

	s.addModel(result)
	return result, nil
}

/**
* loadModel: Loads a model
* @param def et.Json
* @return *Model, error
**/
func (s *Schema) loadModel(def et.Json) (*Model, error) {
	if s.db == nil {
		return nil, errors.New(msg.MSG_DB_IS_NIL)
	}

	if s.db.store == nil {
		return nil, errors.New(msg.MSG_STORE_NOT_DEFINED)
	}

	id := def.Str("id")
	var result *Model
	exists, err := s.db.store.get(id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	result.db = s.db
	result.schema = s
	result.mu = map[string]*sync.RWMutex{}
	result.stores = make(map[string]*store.FileStore, 0)
	result.btrees = make(map[string]*BTree, 0)
	result.onPut = make([]TriggerFnBt, 0)
	result.onRemove = make([]TriggerFnBt, 0)
	result.isDebug = def.Bool("is_debug")

	for _, field := range result.Fields {
		field.from = result
	}

	foreignKeys := def.Json("foreign_keys")
	for name, detail := range result.ForeignKeys {
		detailDef := foreignKeys.Json(name)
		if detailDef.IsEmpty() {
			return nil, fmt.Errorf(msg.MSG_FOREIGN_KEY_NOT_DEFINED, name)
		}

		if err := detail.load(detailDef); err != nil {
			return nil, err
		}
	}

	details := def.Json("details")
	for name, detail := range result.Details {
		detailDef := details.Json(name)
		if detailDef.IsEmpty() {
			return nil, fmt.Errorf(msg.MSG_DETAIL_NOT_DEFINED, name)
		}

		if err := detail.load(detailDef); err != nil {
			return nil, err
		}
	}

	masters := def.Json("masters")
	for name, detail := range result.Masters {
		detailDef := masters.Json(name)
		if detailDef.IsEmpty() {
			return nil, fmt.Errorf(msg.MSG_MASTER_NOT_DEFINED, name)
		}

		if err := detail.load(detailDef); err != nil {
			return nil, err
		}
	}

	rollups := def.Json("rollups")
	for name, detail := range result.Rollups {
		detailDef := rollups.Json(name)
		if detailDef.IsEmpty() {
			return nil, fmt.Errorf(msg.MSG_ROLLUP_NOT_DEFINED, name)
		}

		if err := detail.load(detailDef); err != nil {
			return nil, err
		}
	}

	relations := def.Json("relations")
	for name, detail := range result.Relations {
		detailDef := relations.Json(name)
		if detailDef.IsEmpty() {
			return nil, fmt.Errorf(msg.MSG_RELATION_NOT_DEFINED, name)
		}

		if err := detail.load(detailDef); err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Schema) ToJson() et.Json {
	models := et.Json{}
	for name, model := range s.models {
		models[name] = model.ToJson()
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
* Init: Initializes the schema
* @return error
**/
func (s *Schema) Init() error {
	for _, model := range s.models {
		if err := model.Init(); err != nil {
			return err
		}
	}
	return nil
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

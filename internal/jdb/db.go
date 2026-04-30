package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Config struct {
	Lang                string        `json:"lang"`
	TransactionTTL      time.Duration `json:"transaction_ttl"`
	RelSegSize          int           `json:"rel_seg_size"`
	SyncOnWrite         bool          `json:"sync_on_write"`
	TennantName         string        `json:"tennant_name"`
	TennantPathData     string        `json:"tennant_path_data"`
	Timezone            string        `json:"timezone"`
	MinThresholdCompact int           `json:"min_threshold_compact"`
	TTLTransaction      int           `json:"ttl_transaction"`
	model               *Model        `json:"-"`
}

/**
* DB: Represents a database
**/
type DB struct {
	Name        string             `json:"name"`      // Database name
	Path        string             `json:"path"`      // Path to the database
	Schemas     map[string]*Schema `json:"schemas"`   // Schemas
	IsStrict    bool               `json:"is_strict"` // Is strict mode
	Cache       map[string]*Ttl    `json:"-"`         // Cache
	mu          sync.RWMutex       `json:"-"`         // Mutex
	muCache     sync.RWMutex       `json:"-"`         // Mutex for cache
	config      *Config            `json:"-"`         // Configuration
	errors      *Model             `json:"-"`         // Errors
	transaction *Model             `json:"-"`         // Transaction
	schemas     *Model             `json:"-"`         // Schemas
	cache       *Model             `json:"-"`         // Cache
	node        *Node              `json:"-"`         // Node
}

/**
* NewDb: Creates a new database
* @param path, name string
* @return *DB, error
**/
func NewDb(path, name string) (*DB, error) {
	name = utility.Normalize(name)

	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	path = filepath.Join(path, name)
	result := &DB{
		Name:    name,
		Path:    path,
		Schemas: make(map[string]*Schema, 0),
		mu:      sync.RWMutex{},
	}

	err := loadConfig(result)
	if err != nil {
		return nil, err
	}

	err = loadErrors(result)
	if err != nil {
		return nil, err
	}

	err = loadTransaction(result)
	if err != nil {
		return nil, err
	}

	err = loadSchemas(result)
	if err != nil {
		return nil, err
	}

	err = loadCache(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Init: Initialize the database
* @return error
**/
func (s *DB) Init() error {
	s.mu = sync.RWMutex{}
	s.mu.Lock()
	defer s.mu.Unlock()

	err := loadConfig(s)
	if err != nil {
		return err
	}

	err = loadTransaction(s)
	if err != nil {
		return err
	}

	err = loadSchemas(s)
	if err != nil {
		return err
	}

	return nil
}

/**
* Serialize
* @return []byte, error
**/
func (s *DB) Serialize() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *DB) ToJson() (et.Json, error) {
	definition, err := s.Serialize()
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(definition, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* Save: Save the database
* @param tx *Tx
* @return (*Tx, error)
**/
func (s *DB) Save(tx *Tx) (*Tx, error) {
	item, err := s.ToJson()
	if err != nil {
		return tx, err
	}

	tx, err = s.node.dbs.Upsert(s.Name, item, tx)
	if err != nil {
		return tx, err
	}

	return tx, nil
}

/**
* SetStrict
* @param strict bool
**/
func (s *DB) SetStrict(strict bool) {
	s.IsStrict = strict
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema
**/
func (s *DB) getSchema(name string) *Schema {
	name = utility.Normalize(name)

	s.mu.RLock()
	result, exists := s.Schemas[name]
	s.mu.RUnlock()
	if exists {
		return result
	}

	result = &Schema{
		Database: s.Name,
		Name:     name,
		Models:   make(map[string]*Model, 0),
		db:       s,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Schemas[name] = result

	return result
}

/**
* DeleteSchema: Deletes a schema
* @param name string
* @return error
 */
func (s *DB) DeleteSchema(name string) error {
	name = utility.Normalize(name)

	s.mu.RLock()
	schema, exists := s.Schemas[name]
	s.mu.RUnlock()
	if !exists {
		return errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	err := schema.Empty()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Schemas, name)

	return nil
}

/**
* NewModel: Creates a new model
* @param schema, name	string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) NewModel(schema, name string, isCore bool, version int) (*Model, error) {
	sch := s.getSchema(schema)
	model, err := sch.NewModel(name, isCore, version)
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* GetModel: Returns a model
* @param schema, name string
* @return *Model, error
**/
func (s *DB) GetModel(schema, name string) (*Model, error) {
	schema = utility.Normalize(schema)

	s.mu.RLock()
	schemaObj, exists := s.Schemas[schema]
	s.mu.RUnlock()
	if !exists {
		return nil, errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	return schemaObj.GetModel(name)
}

/**
* DeleteModel: Deletes a model
* @param schema, name string
* @return error
**/
func (s *DB) DeleteModel(schema, name string) error {
	schema = utility.Normalize(schema)
	s.mu.RLock()
	schemaObj, exists := s.Schemas[schema]
	s.mu.RUnlock()
	if !exists {
		return errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	return schemaObj.DeleteModel(name)
}

/**
* Empty: Empties the database
* @return error
**/
func (s *DB) Empty() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, schema := range s.Schemas {
		err := schema.Empty()
		if err != nil {
			return err
		}
	}

	s.Schemas = make(map[string]*Schema, 0)

	return nil
}

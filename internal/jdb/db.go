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
	"github.com/cgalvisleon/josefina/internal/store"
)

const (
	sysDb     = ".catalog"
	sysSchema = ".catalog"
)

type Config struct {
	IsStrict            bool          `json:"is_strict"` // Is strict mode
	Lang                string        `json:"lang"`
	TransactionTTL      time.Duration `json:"transaction_ttl"`
	RelSegSize          int           `json:"rel_seg_size"`
	SyncOnWrite         bool          `json:"sync_on_write"`
	TennantName         string        `json:"tennant_name"`
	TennantPathData     string        `json:"tennant_path_data"`
	Timezone            string        `json:"timezone"`
	MinThresholdCompact int           `json:"min_threshold_compact"`
}

/**
* DB: Represents a database
**/
type DB struct {
	Name        string                   `json:"name"`    // Database name
	Path        string                   `json:"path"`    // Path to the database
	Config      *Config                  `json:"config"`  // Configuration
	Schemas     map[string]*Schema       `json:"schemas"` // Schemas
	Cache       map[string]*Ttl          `json:"-"`       // Cache
	mu          map[string]*sync.RWMutex `json:"-"`       // Mutex
	schemas     *Model                   `json:"-"`       // Schemas
	models      *Model                   `json:"-"`       // Models
	cache       *Model                   `json:"-"`       // Cache
	transaction *Model                   `json:"-"`       // Transaction
	errors      *Model                   `json:"-"`       // Errors
}

/**
* NewDb: Creates a new database
* @param path, name string
* @return *DB, error
**/
func NewDb(path, name string) (*DB, error) {
	name = store.Normalize(name)
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	path = filepath.Join(path, name)
	result := &DB{
		Name: name,
		Path: path,
		Config: &Config{
			IsStrict:            false,
			Lang:                "en",
			TransactionTTL:      10 * time.Second,
			RelSegSize:          1024,
			SyncOnWrite:         false,
			TennantName:         "",
			TennantPathData:     "",
			Timezone:            "America/Bogota",
			MinThresholdCompact: 100,
		},
		Schemas: make(map[string]*Schema, 0),
		Cache:   make(map[string]*Ttl, 0),
		mu:      make(map[string]*sync.RWMutex, 0),
	}

	return result, nil
}

/**
* load: Load the database
* @param node *Node
* @return error
**/
func (s *DB) load() error {
	// Re-initialize unexported fields in deserialized Schema objects.
	for _, schema := range s.Schemas {
		if schema.mu == nil {
			schema.mu = &sync.RWMutex{}
		}
		if schema.models == nil {
			schema.models = make(map[string]*Model)
		}
		schema.db = s
	}
	return s.Init()
}

/**
* Init: Initialize the database
* @return error
**/
func (s *DB) Init() error {
	err := s.loadConfig()
	if err != nil {
		return err
	}

	err = s.loadErrors()
	if err != nil {
		return err
	}

	err = s.loadTransaction()
	if err != nil {
		return err
	}

	err = s.loadSchemas()
	if err != nil {
		return err
	}

	err = s.loadModels()
	if err != nil {
		return err
	}

	err = s.loadCache()
	if err != nil {
		return err
	}

	return nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *DB) ToJson() (et.Json, error) {
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
* Save: Save the database
* @param tx *Tx
* @return (*Tx, error)
**/
func (s *DB) Save() error {
	data, err := s.ToJson()
	if err != nil {
		return err
	}

	_, err = s.node.dbs.
		Insert(data).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* SetStrict
* @param strict bool
**/
func (s *DB) SetStrict(strict bool) {
	s.IsStrict = strict
}

/**
* newSchema: Creates a new schema
* @param db *DB, name string
* @return *Schema
**/
func (s *DB) newSchema(name string) *Schema {
	name = store.Normalize(name)
	if name == "" {
		name = "public"
	}
	result := &Schema{
		Database: s.Name,
		Name:     name,
		models:   make(map[string]*Model, 0),
		db:       s,
		mu:       &sync.RWMutex{},
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Schemas[name] = result

	return result
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema
**/
func (s *DB) getSchema(name string) (*Schema, error) {
	name = store.Normalize(name)
	if name == "" {
		name = "public"
	}
	s.mu.RLock()
	result, exists := s.Schemas[name]
	s.mu.RUnlock()
	if exists {
		return result, nil
	}

	result = s.newSchema(name)
	return result, nil
}

/**
* DeleteSchema: Deletes a schema
* @param name string
* @return error
 */
func (s *DB) DeleteSchema(name string) error {
	name = store.Normalize(name)

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
* newModel: Creates a new model
* @param schema, name	string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) newModel(schema, name string, isCore bool, version int) (*Model, error) {
	sch, err := s.getSchema(schema)
	if err != nil {
		return nil, err
	}

	model, err := sch.newModel(name, isCore, version)
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
	schema = store.Normalize(schema)

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
	schema = store.Normalize(schema)

	s.mu.RLock()
	schemaObj, exists := s.Schemas[schema]
	s.mu.RUnlock()
	if !exists {
		return errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	return schemaObj.DeleteModel(name)
}

/**
* ListSchemas: Returns all schemas in this database.
* @return []*Schema
**/
func (s *DB) ListSchemas() []*Schema {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Schema, 0, len(s.Schemas))
	for _, sc := range s.Schemas {
		result = append(result, sc)
	}
	return result
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

/**
* Define: Defines the model
* @param define DModel
* @return (*Model, error)
**/
func (s *DB) Define(define DModel) (*Model, error) {
	schema := define.Schema
	name := define.Name
	if !utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	isCore := define.IsCore
	version := define.Version
	result, err := s.newModel(schema, name, isCore, version)
	if err != nil {
		return nil, err
	}
	isStrict := define.IsStrict
	result.IsStrict = isStrict

	fields := define.Fields
	for name, field := range fields {
		tpData := TypeData(field.Type)
		defaultValue := field.Default
		_, err := result.DefineField(name, tpData, defaultValue)
		if err != nil {
			return nil, err
		}
	}
	indexes := define.Indexes
	for _, index := range indexes {
		name := index.Name
		tpIndex := index.Type
		if tpIndex == "" {
			tpIndex = "btree"
		}
		_, err := result.DefineIndex(name, TpIndex(tpIndex))
		if err != nil {
			return nil, err
		}
	}
	unique := define.Unique
	for _, index := range unique {
		name := index.Name
		_, err := result.DefineUnique(name)
		if err != nil {
			return nil, err
		}
	}
	required := define.Required
	for _, index := range required {
		name := index.Name
		_, err := result.DefineRequired(name)
		if err != nil {
			return nil, err
		}
	}
	hidden := define.Hidden
	for _, name := range hidden {
		err := result.DefineHidden(name)
		if err != nil {
			return nil, err
		}
	}
	primaryKeys := define.PrimaryKeys
	err = result.DefinePrimaryKeys(primaryKeys...)
	if err != nil {
		return nil, err
	}
	foreignKeys := define.ForeignKeys
	for _, fk := range foreignKeys {
		schema := fk.To.Schema
		toName := fk.To.Name
		to, err := s.GetModel(schema, toName)
		if err != nil {
			return nil, err
		}
		keys := fk.Keys
		onDeleteCascade := fk.OnDeleteCascade
		onUpdateCascade := fk.OnUpdateCascade
		_, err = result.DefineForeignKeys(to, keys, onDeleteCascade, onUpdateCascade)
		if err != nil {
			return nil, err
		}
	}
	details := define.Details
	for name, detail := range details {
		keys := detail.Keys
		version := detail.Version
		_, err = result.DefineDetail(name, keys, version)
		if err != nil {
			return nil, err
		}
	}
	rollups := define.Rollups
	for name, rollup := range rollups {
		schema := rollup.To.Schema
		toName := rollup.To.Name
		to, err := s.GetModel(schema, toName)
		if err != nil {
			return nil, err
		}
		keys := rollup.Keys
		selects := rollup.Selects
		err = result.DefineRollup(name, to, keys, selects)
		if err != nil {
			return nil, err
		}
	}
	relations := define.Relations
	for _, relation := range relations {
		schema := relation.To.Schema
		toName := relation.To.Name
		to, err := s.GetModel(schema, toName)
		if err != nil {
			return nil, err
		}
		keys := relation.Keys
		onDeleteCascade := relation.OnDeleteCascade
		onUpdateCascade := relation.OnUpdateCascade
		err = result.DefineRelation(to, keys, onDeleteCascade, onUpdateCascade)
		if err != nil {
			return nil, err
		}
	}
	calcs := define.Calcs
	for name, definition := range calcs {
		err = result.DefineCalc(name, []byte(definition))
		if err != nil {
			return nil, err
		}
	}
	beforeInsert := define.BeforeInserts
	for _, trigger := range beforeInsert {
		result.AddBeforeInsert(trigger.Name, string(trigger.Definition))
	}
	beforeUpdate := define.BeforeUpdates
	for _, trigger := range beforeUpdate {
		result.AddBeforeUpdate(trigger.Name, string(trigger.Definition))
	}
	beforeDelete := define.BeforeDeletes
	for _, trigger := range beforeDelete {
		result.AddBeforeDelete(trigger.Name, string(trigger.Definition))
	}
	afterInsert := define.AfterInserts
	for _, trigger := range afterInsert {
		result.AddAfterInsert(trigger.Name, string(trigger.Definition))
	}
	afterUpdate := define.AfterUpdates
	for _, trigger := range afterUpdate {
		result.AddAfterUpdate(trigger.Name, string(trigger.Definition))
	}
	afterDelete := define.AfterDeletes
	for _, trigger := range afterDelete {
		result.AddAfterDelete(trigger.Name, string(trigger.Definition))
	}

	return result, nil
}

/**
* Command
* @param cmds []DCmd
* @return (et.Items, error)
 */
func (s *DB) Command(cmds []DCmd) (et.Items, error) {
	result := et.Items{}
	for _, cmd := range cmds {
		if cmd.Insert != nil {
			model, err := s.GetModel(cmd.Insert.Schema, cmd.Insert.Name)
			if err != nil {
				return result, err
			}
			return model.
				Insert(cmd.Insert.Data).
				Exec()
		} else if cmd.Update != nil {
			model, err := s.GetModel(cmd.Update.Schema, cmd.Update.Name)
			if err != nil {
				return result, err
			}
			command := model.Update(cmd.Update.Data)
			for _, condition := range cmd.Update.Where {
				command.Add(&condition)
			}
			return command.Exec()
		} else if cmd.Delete != nil {
			model, err := s.GetModel(cmd.Delete.Schema, cmd.Delete.Name)
			if err != nil {
				return result, err
			}
			command := model.Delete()
			for _, condition := range cmd.Delete.Where {
				command.Add(&condition)
			}
			return command.Exec()
		} else if cmd.Bulk != nil {
			model, err := s.GetModel(cmd.Bulk.Schema, cmd.Bulk.Name)
			if err != nil {
				return result, err
			}
			return model.Bulk(cmd.Bulk.Data).Exec()
		}
	}
	return result, nil
}

/**
**/
func (s *DB) Query(querys []DQuery) (et.Items, error) {
	result := et.NewItems([]et.Json{})
	var tx *Tx
	tx, _ = GetTx(s, nil)
	for _, query := range querys {
		model, err := s.GetModel(query.From.Schema, query.From.Name)
		if err != nil {
			return result, err
		}
		where := From(model)
		for _, condition := range query.Where {
			where.Add(&condition)
		}
		for _, selectField := range query.Selects {
			where.Selects(selectField)
		}
		for _, hiddenField := range query.Hidden {
			where.Hidden(hiddenField)
		}
		for _, orderBy := range query.OrderBy {
			where.orderBy = append(where.orderBy, orderBy)
		}
		where.Limit(query.Limit, query.Page)
		items, err := where.All()
		if err != nil {
			return result, err
		}
		result.Add(items.Result...)
	}
	tx.Commit()
	return result, nil
}

package jdb

import (
	"errors"
	"fmt"
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
	Name        string             `json:"name"`   // Database name
	Path        string             `json:"path"`   // Path to the database
	Config      *Config            `json:"config"` // Configuration
	schemas     map[string]*Schema `json:"-"`      // Schemas
	cache       *Cache             `json:"-"`      // Cache
	mu          *sync.RWMutex      `json:"-"`      // Mutex
	store       *Model             `json:"-"`      // Store
	transaction *Model             `json:"-"`      // Transaction
	errors      *Model             `json:"-"`      // Errors
}

/**
* Init: Initializes the database
* @return error
**/
func (s *DB) Init() error {
	var err error
	s.store, err = s.newModel("", "store", 1, true)
	if err != nil {
		return err
	}

	if err := s.store.Init(); err != nil {
		return err
	}

	s.transaction, err = s.newModel("", "transaction", 1, true)
	if err != nil {
		return err
	}

	if err := s.transaction.Init(); err != nil {
		return err
	}

	s.errors, err = s.newModel("", "errors", 1, true)
	if err != nil {
		return err
	}

	if err := s.errors.Init(); err != nil {
		return err
	}

	return nil
}

/**
* ToJson
* @return et.Json, error
**/
func (s *DB) ToJson() et.Json {
	schemas := []et.Json{}
	for _, schema := range s.schemas {
		schemas = append(schemas, schema.ToJson())
	}

	return et.Json{
		"name":    s.Name,
		"path":    s.Path,
		"config":  s.Config,
		"schemas": s.schemas,
	}
}

/**
* Save: Save the database
* @param tx *Tx
* @return (*Tx, error)
**/
func (s *DB) Save() error {
	if s.store == nil {
		return errors.New(msg.MSG_STORE_NOT_DEFINED)
	}

	err := s.store.putObject(s.Name, s.ToJson())
	if err != nil {
		return err
	}

	return nil
}

/**
* addSchema: Adds a schema to the database
* @param schema *Schema
* @return error
**/
func (s *DB) addSchema(schema *Schema) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.schemas[schema.Name] = schema
	return s.Save()
}

/**
* DeleteSchema: Deletes a schema
* @param name string
* @return error
 */
func (s *DB) DeleteSchema(name string) error {
	name = store.Normalize(name)

	s.mu.RLock()
	defer s.mu.RUnlock()

	schema, exists := s.schemas[name]
	if !exists {
		return errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	err := schema.Empty()
	if err != nil {
		return err
	}

	delete(s.schemas, name)

	return s.Save()
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema, error
**/
func (s *DB) getSchema(name string) (*Schema, bool) {
	name = store.Normalize(name)

	s.mu.RLock()
	defer s.mu.RUnlock()

	schema, exists := s.schemas[name]
	if !exists {
		return nil, false
	}

	return schema, true
}

/**
* newModel: Creates a new model
* @param schema, name	string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) newModel(schema, name string, version int, isCore bool) (*Model, error) {
	sch, exists := s.getSchema(schema)
	if !exists {
		var err error
		sch, err = s.newSchema(schema)
		if err != nil {
			return nil, err
		}
	}

	model, err := sch.newModel(name, s.Path, version, isCore)
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* NewModel: Creates a new model
* @param schema, name string, version int
* @return *Model, error
**/
func (s *DB) NewModel(schema, name string, version int) (*Model, error) {
	result, err := s.newModel(schema, name, version, false)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* GetModel: Returns a model
* @param schema, name string
* @return *Model, error
**/
func (s *DB) GetModel(schema, name string) (*Model, error) {
	sch, exists := s.getSchema(schema)
	if !exists {
		return nil, errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	result, exists := sch.getModel(name)
	if !exists {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return result, nil
}

/**
* DeleteModel: Deletes a model
* @param schema, name string
* @return error
**/
func (s *DB) DeleteModel(schema, name string) error {
	sch, exists := s.getSchema(schema)
	if !exists {
		return errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	err := sch.DeleteModel(name)
	if err != nil {
		return err
	}

	return nil
}

/**
* Empty: Empties the database
* @return error
**/
func (s *DB) Empty() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, schema := range s.schemas {
		err := schema.Empty()
		if err != nil {
			return err
		}
	}

	s.schemas = make(map[string]*Schema, 0)

	return nil
}

/**
* Define: Defines the model
* @param define DModel
* @return (*Model, error)
**/
func (s *DB) Define(define DModel) (*Model, error) {
	if !utility.ValidStr(define.Name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	result, err := s.newModel(define.Schema, define.Name, define.Version, define.IsCore)
	if err != nil {
		return nil, err
	}

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

	masters := define.Masters
	for name, master := range masters {
		toSchema := master.To.Schema
		toName := master.To.Name
		to, err := s.GetModel(toSchema, toName)
		if err != nil {
			return nil, err
		}
		err = result.DefineMaster(name, master.Keys, to, master.ToKeys)
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
				Insert(cmd.Insert.First()).
				Exec()
		} else if cmd.Update != nil {
			model, err := s.GetModel(cmd.Update.Schema, cmd.Update.Name)
			if err != nil {
				return result, err
			}
			command := model.Update(cmd.Update.First())
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
			return model.Bulk(cmd.Bulk.Items).Exec()
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

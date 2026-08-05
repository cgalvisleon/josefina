package jdb

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

/**
* DB: Represents a database
**/
type DB struct {
	Name                string             `json:"name"`                  // Database name
	PathDatabases       string             `json:"path_databases"`        // Path to the databases
	PathWal             string             `json:"path_wal"`              // Path to the wal
	Lang                string             `json:"lang"`                  // Language
	TransactionTTL      time.Duration      `json:"transaction_ttl"`       // Transaction TTL
	RelSegSize          int                `json:"rel_seg_size"`          // Relational segment size
	SyncOnWrite         bool               `json:"sync_on_write"`         // Sync on write
	Timezone            string             `json:"timezone"`              // Timezone
	MinThresholdCompact int                `json:"min_threshold_compact"` // Min threshold compact
	IsStrict            bool               `json:"is_strict"`             // Is strict
	Version             string             `json:"version"`               // Version
	isInit              bool               `json:"-"`                     // Is initialized
	server              *Server            `json:"-"`                     // Server
	schemas             map[string]*Schema `json:"-"`                     // Schemas
	mu                  *sync.RWMutex      `json:"-"`                     // Mutex
	store               *Model             `json:"-"`                     // Store
	cache               *Cache             `json:"-"`                     // Cache
	users               *Users             `json:"-"`                     // Users
}

/**
* newSchema: Creates a new schema
* @param name string
* @return *Schema, error
**/
func (s *DB) newSchema(name string) (*Schema, error) {
	name = store.Normalize(name)
	result := &Schema{
		Database: s.Name,
		Name:     name,
		models:   make(map[string]*Model),
		db:       s,
		mu:       &sync.RWMutex{},
	}
	err := s.addSchema(result)
	if err != nil {
		return nil, err
	}
	return result, nil
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
		"name":                  s.Name,
		"path_databases":        s.PathDatabases,
		"path_wal":              s.PathWal,
		"lang":                  s.Lang,
		"transaction_ttl":       s.TransactionTTL,
		"rel_seg_size":          s.RelSegSize,
		"sync_on_write":         s.SyncOnWrite,
		"timezone":              s.Timezone,
		"min_threshold_compact": s.MinThresholdCompact,
		"is_strict":             s.IsStrict,
		"version":               s.Version,
		"schemas":               schemas,
	}
}

/**
* save: Saves the database
* @return error
**/
func (s *DB) save() error {
	err := s.server.saveDb(s)
	if err != nil {
		return err
	}
	return nil
}

/**
* Init: Initializes the database
* @return error
**/
func (s *DB) init() error {
	for _, schema := range s.schemas {
		if err := schema.Init(); err != nil {
			return err
		}
	}

	s.isInit = true
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
	return s.save()
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

	return s.save()
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

	model, err := sch.newModel(name, version, isCore)
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* loadModel: Loads a model from the JSON definition
* @param schema, name string, version int, isCore bool
* @return *Model, error
**/
func (s *DB) loadModel(schema, name string, version int, isCore bool) (*Model, error) {
	result, err := s.newModel(schema, name, version, isCore)
	if err != nil {
		return nil, err
	}

	if err := result.Init(); err != nil {
		return nil, err
	}

	return result, nil
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
	if exists {
		return result, nil
	}

	if s.IsStrict {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	result, err := sch.newModel(name, 1, false)
	if err != nil {
		return nil, err
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
* defineUser: Defines the user
* @param define et.Json
* @return et.Json, error
**/
func (s *DB) defineUser(define et.Json) (et.Json, error) {
	username := define.Str("username")
	if !utility.ValidStr(username, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	password := define.Str("password")
	if !utility.ValidStr(password, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "password")
	}

	user, err := s.newUser(username, password)
	if err != nil {
		return nil, err
	}

	return user.ToJson(), nil
}

/**
* DefineSchema: Defines the schema
* @param define et.Json
* @return et.Json, error
**/
func (s *DB) defineSchema(define et.Json) (et.Json, error) {
	name := define.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	schema, err := s.newSchema(name)
	if err != nil {
		return nil, err
	}

	return schema.ToJson(), nil
}

/**
* defineModel: Defines the model
* @param define et.Json
* @return et.Json, error
**/
func (s *DB) defineModel(define et.Json) (et.Json, error) {
	name := define.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	schema := define.Str("schema")
	if !utility.ValidStr(schema, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	version := define.ValInt(1, "version")
	result, err := s.newModel(schema, name, version, false)
	if err != nil {
		return nil, err
	}

	fields := define.Json("fields")
	for name := range fields {
		field := fields.Json(name)
		tpData := TypeData(field.Str("type"))
		defaultValue := field.ValAny("default")
		_, err := result.DefineField(name, tpData, defaultValue)
		if err != nil {
			return nil, err
		}
	}

	indexes := define.ArrayJson("indexes")
	for _, index := range indexes {
		name := index.Str("name")
		tpIndex := index.Str("type")
		if tpIndex == "" {
			tpIndex = "btree"
		}
		_, err := result.DefineIndex(name, TpIndex(tpIndex))
		if err != nil {
			return nil, err
		}
	}

	unique := define.ArrayJson("unique")
	for _, index := range unique {
		name := index.Str("name")
		_, err := result.DefineUnique(name)
		if err != nil {
			return nil, err
		}
	}

	required := define.ArrayJson("required")
	for _, index := range required {
		name := index.Str("name")
		_, err := result.DefineRequired(name)
		if err != nil {
			return nil, err
		}
	}

	hidden := define.ArrayStr("hidden")
	for _, name := range hidden {
		err := result.DefineHidden(name)
		if err != nil {
			return nil, err
		}
	}

	primaryKeys := define.ArrayStr("primary_keys")
	err = result.DefinePrimaryKeys(primaryKeys...)
	if err != nil {
		return nil, err
	}

	foreignKeys := define.ArrayJson("foreign_keys")
	for _, fk := range foreignKeys {
		schema := fk.Str("to.schema")
		toName := fk.Str("to.name")
		to, err := s.GetModel(schema, toName)
		if err != nil {
			return nil, err
		}
		keys := fk.MapStr("keys")
		onDeleteCascade := fk.Bool("on_delete_cascade")
		onUpdateCascade := fk.Bool("on_update_cascade")
		_, err = result.DefineForeignKeys(to, keys, onDeleteCascade, onUpdateCascade)
		if err != nil {
			return nil, err
		}
	}

	details := define.Json("details")
	for name := range details {
		detail := details.Json(name)
		keys := detail.MapStr("keys")
		version := detail.ValInt(1, "version")
		_, err = result.DefineDetail(name, keys, version)
		if err != nil {
			return nil, err
		}
	}

	masters := define.Json("masters")
	for name := range masters {
		master := masters.Json(name)
		toSchema := master.Str("to.schema")
		toName := master.Str("to.name")
		to, err := s.GetModel(toSchema, toName)
		if err != nil {
			return nil, err
		}
		keys := master.MapStr("keys")
		toKeys := master.MapStr("to_keys")
		err = result.DefineMaster(name, keys, to, toKeys)
		if err != nil {
			return nil, err
		}
	}

	rollups := define.Json("rollups")
	for name := range rollups {
		rollup := rollups.Json(name)
		schema := rollup.Str("to.schema")
		toName := rollup.Str("to.name")
		to, err := s.GetModel(schema, toName)
		if err != nil {
			return nil, err
		}
		keys := rollup.MapStr("keys")
		selects := rollup.ArrayStr("selects")
		err = result.DefineRollup(name, to, keys, selects)
		if err != nil {
			return nil, err
		}
	}

	relations := define.Json("relations")
	for name := range relations {
		relation := relations.Json(name)
		schema := relation.Str("to.schema")
		toName := relation.Str("to.name")
		to, err := s.GetModel(schema, toName)
		if err != nil {
			return nil, err
		}
		keys := relation.MapStr("keys")
		onDeleteCascade := relation.Bool("on_delete_cascade")
		onUpdateCascade := relation.Bool("on_update_cascade")
		err = result.DefineRelation(to, keys, onDeleteCascade, onUpdateCascade)
		if err != nil {
			return nil, err
		}
	}

	calcs := define.MapStr("calcs")
	for name, definition := range calcs {
		err = result.DefineCalc(name, definition)
		if err != nil {
			return nil, err
		}
	}

	beforeInsert := define.ArrayJson("before_inserts")
	for _, trigger := range beforeInsert {
		name := trigger.Str("name")
		definition := trigger.Str("definition")
		result.AddBeforeInsert(name, definition)
	}

	beforeUpdate := define.ArrayJson("before_updates")
	for _, trigger := range beforeUpdate {
		name := trigger.Str("name")
		definition := trigger.Str("definition")
		result.AddBeforeUpdate(name, definition)
	}

	beforeDelete := define.ArrayJson("before_deletes")
	for _, trigger := range beforeDelete {
		name := trigger.Str("name")
		definition := trigger.Str("definition")
		result.AddBeforeDelete(name, string(definition))
	}

	afterInsert := define.ArrayJson("after_inserts")
	for _, trigger := range afterInsert {
		name := trigger.Str("name")
		definition := trigger.Str("definition")
		result.AddAfterInsert(name, definition)
	}

	afterUpdate := define.ArrayJson("after_updates")
	for _, trigger := range afterUpdate {
		name := trigger.Str("name")
		definition := trigger.Str("definition")
		result.AddAfterUpdate(name, definition)
	}

	afterDelete := define.ArrayJson("after_deletes")
	for _, trigger := range afterDelete {
		name := trigger.Str("name")
		definition := trigger.Str("definition")
		result.AddAfterDelete(name, definition)
	}

	if err := result.Init(); err != nil {
		return nil, err
	}

	return result.ToJson(), nil
}

/**
* describeUser: Describes the user
* @param describe et.Json
* @return et.Json, error
**/
func (s *DB) describeUser(describe et.Json) (et.Json, error) {
	name := describe.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	user, err := s.getUser(name)
	if err != nil {
		return nil, err
	}

	return user.ToJson(), nil
}

/**
* describeSchema: Describes the schema
* @param describe et.Json
* @return et.Json, error
**/
func (s *DB) describeSchema(describe et.Json) (et.Json, error) {
	name := describe.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	schema, exists := s.getSchema(name)
	if !exists {
		return nil, errors.New(msg.MSG_SCHEMA_NOT_FOUND)
	}

	return schema.ToJson(), nil
}

/**
* describeModel: Describes the model
* @param describe et.Json
* @return et.Json, error
**/
func (s *DB) describeModel(describe et.Json) (et.Json, error) {
	schema := describe.Str("schema")
	if !utility.ValidStr(schema, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	name := describe.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	model, err := s.GetModel(schema, name)
	if err != nil {
		return nil, err
	}

	return model.ToJson(), nil
}

/**
* insertQuery: Inserts a record
* @param define et.Json
* @return et.Json, error
**/
func (s *DB) insertQuery(define et.Json) ([]et.Json, error) {
	schema := define.Str("schema")
	if !utility.ValidStr(schema, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	name := define.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	model, err := s.GetModel(schema, name)
	if err != nil {
		return []et.Json{}, err
	}

	data := define.Json("data")
	cmd := model.Insert(data)
	result, err := cmd.
		Exec()
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}

/**
* updateQuery: Updates a record
* @param define et.Json
* @return []et.Json, error
**/
func (s *DB) updateQuery(define et.Json) ([]et.Json, error) {
	schema := define.Str("schema")
	if !utility.ValidStr(schema, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	name := define.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	model, err := s.GetModel(schema, name)
	if err != nil {
		return []et.Json{}, err
	}

	data := define.Json("data")
	cmd := model.Update(data)
	wheres := define.ArrayJson("where")
	for _, where := range wheres {
		condition := et.ToCondition(where)
		for _, cond := range condition {
			cmd.Add(cond)
		}
	}

	result, err := cmd.
		Exec()
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}

/**
* deleteQuery: Deletes a record
* @param define et.Json
* @return []et.Json, error
**/
func (s *DB) deleteQuery(define et.Json) ([]et.Json, error) {
	schema := define.Str("schema")
	if !utility.ValidStr(schema, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	name := define.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	model, err := s.GetModel(schema, name)
	if err != nil {
		return []et.Json{}, err
	}

	cmd := model.Delete()
	wheres := define.ArrayJson("where")
	for _, where := range wheres {
		condition := et.ToCondition(where)
		for _, cond := range condition {
			cmd.Add(cond)
		}
	}

	result, err := cmd.
		Exec()
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}

/**
* upsertQuery: Upserts a record
* @param define et.Json
* @return []et.Json, error
**/
func (s *DB) upsertQuery(define et.Json) ([]et.Json, error) {
	schema := define.Str("schema")
	if !utility.ValidStr(schema, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	name := define.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	model, err := s.GetModel(schema, name)
	if err != nil {
		return []et.Json{}, err
	}

	data := define.Json("data")
	cmd := model.Upsert(data)
	wheres := define.ArrayJson("where")
	for _, where := range wheres {
		condition := et.ToCondition(where)
		for _, cond := range condition {
			cmd.Add(cond)
		}
	}

	result, err := cmd.
		Exec()
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}

/**
* bulkQuery: Bulk inserts a record
* @param define et.Json
* @return []et.Json, error
**/
func (s *DB) bulkQuery(define et.Json) ([]et.Json, error) {
	schema := define.Str("schema")
	if !utility.ValidStr(schema, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "schema")
	}

	name := define.Str("name")
	if !utility.ValidStr(name, 1, []string{}) {
		return []et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	model, err := s.GetModel(schema, name)
	if err != nil {
		return []et.Json{}, err
	}

	data := define.ArrayJson("data")
	cmd := model.Bulk(data)
	result, err := cmd.
		Exec()
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}

/**
* jSystem: Executes a system command
* @param params et.Json
* @return et.Items, error
**/
func (s *DB) jSystem(params et.Json) (et.Items, error) {
	define := params.ArrayJson("define")
	describe := params.ArrayJson("describe")

	jobs := []queryJob{
		{defineQuery, define},
		{describeQuery, describe},
	}

	result, err := runQueryJobs(s, jobs)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* jQuery: Executes a query
* @param query string
* @return et.Items, error
**/
func (s *DB) jQuery(params et.Json) (et.Items, error) {
	query := params.ArrayJson("query")

	jobs := []queryJob{
		{execQuery, query},
	}

	result, err := runQueryJobs(s, jobs)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* jCommand: Executes a command
* @param params et.Json
* @return et.Items, error
**/
func (s *DB) jCommand(params et.Json) (et.Items, error) {
	insert := params.ArrayJson("insert")
	update := params.ArrayJson("update")
	delete := params.ArrayJson("delete")
	bulk := params.ArrayJson("bulk")

	jobs := []queryJob{
		{insertQuery, insert},
		{updateQuery, update},
		{deleteQuery, delete},
		{bulkQuery, bulk},
	}

	result, err := runQueryJobs(s, jobs)
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

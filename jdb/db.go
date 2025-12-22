package jdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

var databases []*DB

func init() {
	databases = []*DB{}
}

type DB struct {
	Id         string   `json:"id"`
	Name       string   `json:"name"`
	Models     []*Model `json:"models"`
	UseCore    bool     `json:"use_core"`
	Connection et.Json  `json:"connection"`
	Language   string   `json:"language"`
	Db         *sql.DB  `json:"-"`
	driver     Driver   `json:"-"`
}

/**
* indexDatabase
* @param name string
* @return int
**/
func indexDatabase(name string) int {
	return slices.IndexFunc(databases, func(db *DB) bool { return db.Name == name })
}

/**
* getDatabase
* @param id, name, driver string, userCore bool, params Connection
* @return (*DB, error)
**/
func getDatabase(id, name, driver string, userCore bool, params et.Json) (*DB, error) {
	idx := indexDatabase(name)
	if idx != -1 {
		return databases[idx], nil
	}

	if _, ok := drivers[driver]; !ok {
		return nil, fmt.Errorf(MSG_DRIVER_NOT_FOUND, driver)
	}

	id = reg.TagULID("db", id)
	result := &DB{
		Id:         id,
		Name:       name,
		Models:     make([]*Model, 0),
		UseCore:    userCore,
		Connection: params,
		Language:   "en",
	}
	result.driver = drivers[driver](result)
	err := result.load()
	if err != nil {
		return nil, err
	}

	databases = append(databases, result)

	return result, nil
}

/**
* Serialize
* @return []byte, error
**/
func (s *DB) Serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *DB) ToJson() et.Json {
	bt, err := s.Serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* load
* @return error
**/
func (s *DB) load() error {
	if s.driver == nil {
		return fmt.Errorf(MSG_DRIVER_NOT_FOUND)
	}

	db, err := s.driver.Connect(s)
	if err != nil {
		return err
	}

	s.Db = db
	if s.UseCore {
		err := s.initCore()
		if err != nil {
			logs.Panic(err)
		}
	}
	loadMsg(s.Language)

	return nil
}

/**
* idxModel
* @param name string
* @return int
**/
func (s *DB) idxModel(name string) int {
	return slices.IndexFunc(s.Models, func(model *Model) bool { return model.Name == name })
}

/**
* newModel
* @param schema, name string
* @return *Model
**/
func (s *DB) newModel(schema, name string, version int) (*Model, error) {
	if !utility.ValidStr(name, 0, []string{}) {
		return nil, fmt.Errorf(MSG_ATTRIBUTE_REQUIRED, name)
	}

	if !utility.ValidStr(schema, 0, []string{}) {
		return nil, fmt.Errorf(MSG_ATTRIBUTE_REQUIRED, schema)
	}

	if version <= 0 {
		version = 1
	}

	result := &Model{
		DB:            s,
		Schema:        schema,
		Name:          name,
		Table:         name,
		Columns:       make([]*Column, 0),
		PrimaryKeys:   make([]string, 0),
		Unique:        make([]string, 0),
		Indexes:       make([]string, 0),
		Required:      make([]string, 0),
		Hidden:        make([]string, 0),
		Master:        make(map[string]*Detail),
		Details:       make(map[string]*Detail),
		Rollups:       make(map[string]*Detail),
		Relations:     make(map[string]*Detail),
		BeforeInserts: make([]*Trigger, 0),
		BeforeUpdates: make([]*Trigger, 0),
		BeforeDeletes: make([]*Trigger, 0),
		AfterInserts:  make([]*Trigger, 0),
		AfterUpdates:  make([]*Trigger, 0),
		AfterDeletes:  make([]*Trigger, 0),
		Version:       version,
		beforeInserts: make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
	}
	s.Models = append(s.Models, result)
	return result, nil
}

/**
* loadModel
* @param name string
* @return *Model
**/
func (s *DB) loadModel(name string) (*Model, error) {
	if models == nil {
		return nil, ErrModelNotFound
	}

	items, err := models.
		Where(Eq("A.name", name)).
		One()
	if err != nil {
		return nil, err
	}

	if !items.Ok {
		return nil, ErrModelNotFound
	}

	scr, err := items.Byte("definition")
	if err != nil {
		return nil, err
	}

	var result *Model
	err = json.Unmarshal(scr, &result)
	if err != nil {
		return nil, err
	}

	result.DB = s
	result.beforeInserts = make([]TriggerFunction, 0)
	result.beforeUpdates = make([]TriggerFunction, 0)
	result.beforeDeletes = make([]TriggerFunction, 0)
	result.afterInserts = make([]TriggerFunction, 0)
	result.afterUpdates = make([]TriggerFunction, 0)
	result.afterDeletes = make([]TriggerFunction, 0)
	s.Models = append(s.Models, result)
	return result, nil
}

/**
* GetModel
* @param name string
* @return *Model
**/
func (s *DB) GetModel(name string) (*Model, error) {
	idx := s.idxModel(name)
	if idx != -1 {
		return s.Models[idx], nil
	}

	return s.loadModel(name)
}

/**
* NewModel
* @params schema, name string, version int
* @return *Model
**/
func (s *DB) NewModel(schema, name string, version int) (*Model, error) {
	idx := s.idxModel(name)
	if idx != -1 {
		result := s.Models[idx]
		if result.Version < version {
			return s.Mutate(result)
		}

		return result, nil
	}

	return s.newModel(schema, name, version)
}

/**
* DeleteModel
* @param name string
* @return error
**/
func (s *DB) DeleteModel(name string) error {
	return s.deleteModel(name)
}

/**
* Load
* @param model *Model
* @return et.Item, error
**/
func (s *DB) Load(model *Model) (et.Item, error) {
	if s.driver == nil {
		return et.Item{}, fmt.Errorf(MSG_DRIVER_NOT_FOUND)
	}

	if model.IsDebug {
		logs.Debugf("load:%s", model.ToJson().ToEscapeHTML())
	}

	sql, err := s.driver.Load(model)
	if err != nil {
		return et.Item{}, err
	}

	result, err := s.SqlTx(nil, sql)
	if err != nil {
		return et.Item{}, err
	}

	return result.First(), nil
}

/**
* Mutate
* @param model *Model
* @return *Model, error
**/
func (s *DB) Mutate(model *Model) (*Model, error) {
	if s.driver == nil {
		return nil, fmt.Errorf(MSG_DRIVER_NOT_FOUND)
	}

	if model.IsDebug {
		logs.Debugf("load:%s", model.ToJson().ToEscapeHTML())
	}

	sql, err := s.driver.Mutate(model)
	if err != nil {
		return nil, err
	}

	_, err = s.SqlTx(nil, sql)
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* Command
* @param command *Command
* @return et.Items, error
**/
func (s *DB) Command(command *Cmd) (et.Items, error) {
	if s.driver == nil {
		return et.Items{}, fmt.Errorf(MSG_DRIVER_NOT_FOUND)
	}

	if command.IsDebug {
		logs.Debugf("command:%s", command.ToJson().ToEscapeHTML())
	}

	sql, err := s.driver.Command(command)
	if err != nil {
		return et.Items{}, err
	}

	return s.SqlTx(command.tx, sql)
}

/**
* Ql
* @param query *Ql
* @return et.Items, error
**/
func (s *DB) Query(query *Ql) (et.Items, error) {
	if s.driver == nil {
		return et.Items{}, fmt.Errorf(MSG_DRIVER_NOT_FOUND)
	}

	if query.IsDebug {
		logs.Debugf("command:%s", query.ToJson().ToEscapeHTML())
	}

	sql, err := s.driver.Query(query)
	if err != nil {
		return et.Items{}, err
	}

	return s.SqlTx(query.tx, sql)
}

/**
* Define
* @return *Model, error
**/
func (s *DB) Define(definition et.Json) (*Model, error) {
	schema := definition.String("schema")
	if !utility.ValidStr(schema, 0, []string{}) {
		return nil, fmt.Errorf(MSG_ATTRIBUTE_REQUIRED, "schema")
	}

	name := definition.String("name")
	if !utility.ValidStr(name, 0, []string{}) {
		return nil, fmt.Errorf(MSG_ATTRIBUTE_REQUIRED, "name")
	}

	version := definition.ValInt(1, "version")
	result, err := s.NewModel(schema, name, version)
	if err != nil {
		return nil, err
	}

	isCore := definition.Bool("is_core")
	if isCore {
		result.IsCore = true
	}

	columns := definition.ArrayJson("columns")
	for _, column := range columns {
		name := column.String("name")
		tpColumn := column.String("tp_column")
		tpData := column.String("tp_data")
		hidden := column.Bool("hidden")
		defaultValue := column.String("default_value")
		definition, err := column.Byte("definition")
		if err != nil {
			return nil, err
		}
		_, err = result.defineColumn(name, TypeColumn(tpColumn), TypeData(tpData), hidden, defaultValue, definition)
		if err != nil {
			return nil, err
		}
	}

	source := definition.String("source")
	if utility.ValidStr(source, 0, []string{}) {
		err := result.DefineSourceField(source)
		if err != nil {
			return nil, err
		}
	}

	indexField := definition.String("index_field")
	if utility.ValidStr(indexField, 0, []string{}) {
		err := result.DefineIndexField(indexField)
		if err != nil {
			return nil, err
		}
	}

	primaryKeys := definition.ArrayStr("primary_keys")
	for _, primaryKey := range primaryKeys {
		result.DefinePrimaryKeys(primaryKey)
	}

	unique := definition.ArrayStr("unique")
	for _, unique := range unique {
		result.DefineUnique(unique)
	}

	indexes := definition.ArrayStr("indexes")
	for _, index := range indexes {
		err := result.DefineIndex(index)
		if err != nil {
			return nil, err
		}
	}

	required := definition.ArrayStr("required")
	for _, required := range required {
		result.DefineRequired(required)
	}

	hidden := definition.ArrayStr("hidden")
	for _, hidden := range hidden {
		result.DefineHidden(hidden)
	}

	details := definition.Json("details")
	for name := range details {
		jkeys := details.Json(name)
		keys := make(map[string]string, 0)
		for pk := range jkeys {
			fk := jkeys.Str(pk)
			keys[pk] = fk
		}
		_, err = result.DefineDetail(name, keys)
		if err != nil {
			return nil, err
		}
	}

	rollups := definition.Json("rollups")
	for name := range rollups {
		detail := rollups.Json(name)
		from := detail.String("from", "name")
		jkeys := detail.Json("keys")
		keys := make(map[string]string, 0)
		for pk := range jkeys {
			fk := jkeys.Str(pk)
			keys[pk] = fk
		}
		selects := detail.ArrayStr("selects")
		err = result.DefineRollup(name, from, keys, selects)
		if err != nil {
			return nil, err
		}
	}

	relations := definition.Json("relations")
	for name := range relations {
		detail := rollups.Json(name)
		jkeys := detail.Json("keys")
		keys := make(map[string]string, 0)
		for pk := range jkeys {
			fk := jkeys.Str(pk)
			keys[pk] = fk
		}
		err = result.DefineRelation(name, keys)
		if err != nil {
			return nil, err
		}
	}

	isLocked := definition.Bool("is_locked")
	result.IsLocked = isLocked
	isDebug := definition.Bool("is_debug")
	result.IsDebug = isDebug

	build := definition.Bool("build")
	if !build {
		return result, nil
	}

	if err = result.Init(); err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Select
* @param query et.Json
* @return et.Items, error
**/
func (s *DB) Select(query et.Json) (et.Items, error) {
	bt, err := query.ToByte()
	if err != nil {
		return et.Items{}, err
	}

	var ql *Ql
	err = json.Unmarshal(bt, &ql)
	if err != nil {
		return et.Items{}, err
	}

	ql.DB = s
	result, err := ql.All()
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* Insert
* @param query et.Json
* @return et.Items, error
**/
func (s *DB) Insert(query et.Json) (et.Items, error) {
	from := query.String("from")
	if !utility.ValidStr(from, 0, []string{}) {
		return et.Items{}, fmt.Errorf(MSG_FROM_REQUIRED)
	}

	model, err := s.GetModel(from)
	if err != nil {
		return et.Items{}, err
	}

	data := query.Json("data")
	if len(data) == 0 {
		return et.Items{}, fmt.Errorf(MSG_DATA_REQUIRED)
	}

	cmd := model.Insert(data)
	result, err := cmd.Exec()
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* Update
* @param query et.Json
* @return et.Items, error
**/
func (s *DB) Update(query et.Json) (et.Items, error) {
	from := query.String("from")
	if !utility.ValidStr(from, 0, []string{}) {
		return et.Items{}, fmt.Errorf(MSG_FROM_REQUIRED)
	}

	model, err := s.GetModel(from)
	if err != nil {
		return et.Items{}, err
	}

	data := query.Json("data")
	if len(data) == 0 {
		return et.Items{}, fmt.Errorf(MSG_DATA_REQUIRED)
	}

	where := query.ArrayJson("where")
	if len(where) == 0 {
		return et.Items{}, fmt.Errorf(MSG_DATA_REQUIRED)
	}

	cmd := model.Update(data)
	cmd.WhereByJson(where)
	result, err := cmd.Exec()
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* Delete
* @param query et.Json
* @return et.Items, error
**/
func (s *DB) Delete(query et.Json) (et.Items, error) {
	from := query.String("from")
	if !utility.ValidStr(from, 0, []string{}) {
		return et.Items{}, fmt.Errorf(MSG_FROM_REQUIRED)
	}

	model, err := s.GetModel(from)
	if err != nil {
		return et.Items{}, err
	}

	where := query.ArrayJson("where")
	if len(where) == 0 {
		return et.Items{}, fmt.Errorf(MSG_DATA_REQUIRED)
	}

	cmd := model.Delete()
	cmd.WhereByJson(where)
	result, err := cmd.Exec()
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

/**
* Upsert
* @param query et.Json
* @return et.Items, error
**/
func (s *DB) Upsert(query et.Json) (et.Items, error) {
	from := query.String("from")
	if !utility.ValidStr(from, 0, []string{}) {
		return et.Items{}, fmt.Errorf(MSG_FROM_REQUIRED)
	}

	model, err := s.GetModel(from)
	if err != nil {
		return et.Items{}, err
	}

	data := query.Json("data")
	if len(data) == 0 {
		return et.Items{}, fmt.Errorf(MSG_DATA_REQUIRED)
	}

	where := query.ArrayJson("where")
	if len(where) == 0 {
		return et.Items{}, fmt.Errorf(MSG_DATA_REQUIRED)
	}

	cmd := model.Upsert(data)
	cmd.WhereByJson(where)
	result, err := cmd.Exec()
	if err != nil {
		return et.Items{}, err
	}

	return result, nil
}

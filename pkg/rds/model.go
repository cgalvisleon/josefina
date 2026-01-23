package rds

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/cgalvisleon/josefina/pkg/store"
)

var (
	errorRecordNotFound      = errors.New(msg.MSG_RECORD_NOT_FOUND)
	errorPrimaryKeysNotFound = errors.New(msg.MSG_PRIMARY_KEYS_NOT_FOUND)
	errorFieldNotFound       = errors.New(msg.MSG_FIELD_NOT_FOUND)
	models                   *Model
)

/**
* initModels: Initializes the models model
* @return error
**/
func initModels() error {
	if models != nil {
		return nil
	}

	db, err := getDb(packageName)
	if err != nil {
		return err
	}

	models, err = db.newModel("", "models", true, 1)
	if err != nil {
		return err
	}
	if err := models.init(); err != nil {
		return err
	}

	return nil
}

type From struct {
	Database string            `json:"database"`
	Schema   string            `json:"schema"`
	Name     string            `json:"name"`
	Fields   map[string]*Field `json:"fields"`
	IsStrict bool              `json:"is_strict"`
	Host     string            `json:"-"`
}

/**
* getJid: Gets the jid
* @return string
**/
func (s *From) getJid() string {
	return reg.GenULID(s.Name)
}

type Model struct {
	*From         `json:"from"`
	Path          string             `json:"path"`
	Indexes       []string           `json:"indexes"`
	PrimaryKeys   []string           `json:"primary_keys"`
	Unique        []string           `json:"unique"`
	Required      []string           `json:"required"`
	Hidden        []string           `json:"hidden"`
	References    map[string]*Detail `json:"references"`
	Details       map[string]*Detail `json:"details"`
	Rollups       map[string]*Detail `json:"rollups"`
	Relations     map[string]*Detail `json:"relations"`
	Calcs         map[string][]byte  `json:"calcs"`
	BeforeInserts []*Trigger         `json:"before_inserts"`
	BeforeUpdates []*Trigger         `json:"before_updates"`
	BeforeDeletes []*Trigger         `json:"before_deletes"`
	AfterInserts  []*Trigger         `json:"after_inserts"`
	AfterUpdates  []*Trigger         `json:"after_updates"`
	AfterDeletes  []*Trigger         `json:"after_deletes"`
	Version       int                `json:"version"`
	IsCore        bool               `json:"is_core"`
	IsInit        bool               `json:"-"`
	isDebug       bool               `json:"-"`
	// db            *DB                         `json:"-"`
	stores   map[string]*store.FileStore `json:"-"`
	triggers map[string]*Vm              `json:"-"`
	changed  bool                        `json:"-"`
}

/**
* newModel
* @param database, schema, name string, isCore bool, version int
* @return *Model
**/
func newModel(database, schema, name string, isCore bool, version int) (*Model, error) {
	if !utility.ValidStr(database, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "database")
	}
	if !utility.ValidStr(name, 1, []string{}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	database = utility.Normalize(database)
	schema = utility.Normalize(schema)
	name = utility.Normalize(name)
	path := envar.GetStr("DATA_PATH", "./data")
	path = fmt.Sprintf("%s/%s", path, database)
	path = strs.Append(path, schema, "/")
	path = fmt.Sprintf("%s/%s", path, name)
	result := &Model{
		From: &From{
			Database: database,
			Schema:   schema,
			Name:     name,
			Fields:   make(map[string]*Field, 0),
		},
		Path:          path,
		Indexes:       make([]string, 0),
		PrimaryKeys:   make([]string, 0),
		Unique:        make([]string, 0),
		Required:      make([]string, 0),
		Hidden:        make([]string, 0),
		References:    make(map[string]*Detail, 0),
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
		triggers:      make(map[string]*Vm, 0),
	}
	_, err := result.defineIndexField()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* serialize
* @return []byte, error
**/
func (s *Model) serialize() ([]byte, error) {
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
func (s *Model) toJson() (et.Json, error) {
	definition, err := s.serialize()
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
* prepared: Prepares the model
* @return error
**/
func (s *Model) load() error {
	if len(s.Indexes) == 0 {
		return errors.New(msg.MSG_INDEX_NOT_DEFINED)
	}

	for _, name := range s.Indexes {
		fStore, err := store.Open(s.Path, name, s.isDebug)
		if err != nil {
			return err
		}
		s.stores[name] = fStore
	}

	s.IsInit = true
	return nil
}

/**
* init: Initializes the model
* @return error
**/
func (s *Model) init() error {
	if s.IsInit {
		return nil
	}

	err := s.load()
	if err != nil {
		return err
	}

	if node != nil {
		s.Host = node.host
	}
	return nil
}

/**
* store: Returns the index
* @param name string
* @return *store.FileStore, bool
**/
func (s *Model) store(name string) (*store.FileStore, error) {
	result, ok := s.stores[name]
	if !ok {
		return nil, errors.New(msg.MSG_STORE_NOT_DEFINED)
	}

	return result, nil
}

/**
* source: Returns the source
* @return *store.FileStore, error
**/
func (s *Model) source() (*store.FileStore, error) {
	result, err := s.store(INDEX)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* key: Returns the key of the model
* @return string
**/
func (s *Model) key() string {
	return modelKey(s.Database, s.Schema, s.Name)
}

/**
* count: Counts the model
* @return int, error
**/
func (s *Model) count() (int, error) {
	result, err := s.source()
	if err != nil {
		return 0, err
	}

	return result.Count(), nil
}

/**
* genKey: Returns a new key for the model
* @return string
**/
func (s *Model) genKey() string {
	return reg.GenUUId(s.Name)
}

/**
* get: Gets the model
* @param key string, dest any
* @return bool, error
**/
func (s *Model) get(key string, dest any) (bool, error) {
	source, err := s.source()
	if err != nil {
		return false, err
	}

	exists, err := source.Get(key, &dest)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	return true, nil
}

/**
* getObjet: Gets the model as object
* @param key string
* @return et.Json, error
**/
func (s *Model) getObjet(key string, dest et.Json) (bool, error) {
	return s.get(key, &dest)
}

/**
* getIndex: Gets the index
* @param field, key string, dest map[string]bool
* @return bool, error
**/
func (s *Model) getIndex(field, key string, dest map[string]bool) (bool, error) {
	index, err := s.store(field)
	if err != nil {
		return false, err
	}

	exists, err := index.Get(key, &dest)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	return true, nil
}

/**
* isExisted: Check if index exists in model
* @param name string, key string
* @return bool, error
**/
func (s *Model) isExisted(name, key string) (bool, error) {
	source, err := s.store(name)
	if err != nil {
		return false, err
	}

	return source.IsExist(key), nil
}

/**
* put: Puts the model
* @param idx string, valu any
* @return error
**/
func (s *Model) put(key string, value any) error {
	source, err := s.source()
	if err != nil {
		return err
	}

	err = source.Put(key, value)
	if err != nil {
		return err
	}
	if !s.IsCore {
		return setRecord(s.Schema, s.Name, key)
	}

	return nil
}

/**
* remove: Removes the model
* @param key string
* @return error
**/
func (s *Model) remove(key string) error {
	source, err := s.source()
	if err != nil {
		return err
	}

	_, err = source.Delete(key)
	if err != nil {
		return err
	}
	if !s.IsCore {
		return deleteRecord(s.Schema, s.Name, key)
	}

	return nil
}

/**
* putIndex
* @param store *store.FileStore, id string, idx any
* @return error
**/
func (s *Model) putIndex(store *store.FileStore, id string, idx any) error {
	result := map[string]bool{}
	exists, err := store.Get(id, &result)
	if err != nil {
		return err
	}

	if !exists {
		result = map[string]bool{}
	}

	key := fmt.Sprintf("%v", idx)
	_, ok := result[key]
	if ok {
		return nil
	}

	result[key] = true
	err = store.Put(id, result)
	if err != nil {
		return err
	}

	return nil
}

/**
* removeIndex
* @param store *store.FileStore, id string, idx any
* @return error
**/
func (s *Model) removeIndex(store *store.FileStore, id string, idx any) error {
	result := map[string]bool{}
	exists, err := store.Get(id, &result)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	key := fmt.Sprintf("%v", idx)
	_, ok := result[key]
	if !ok {
		return nil
	}

	delete(result, key)
	if len(result) == 0 {
		_, err = store.Delete(id)
		if err != nil {
			return err
		}
		return nil
	}

	err = store.Put(id, result)
	if err != nil {
		return err
	}

	return nil
}

/**
* putData: Puts the model
* @param idx string, data et.Json
* @return error
**/
func (s *Model) putData(idx string, data et.Json) error {
	data[INDEX] = idx
	for _, name := range s.Indexes {
		key := fmt.Sprintf("%v", data[name])
		if key == "" {
			continue
		}

		source := s.stores[name]
		if name == INDEX {
			err := source.Put(key, data)
			if err != nil {
				return err
			}
			if !s.IsCore {
				return setRecord(s.Schema, s.Name, key)
			}
		} else {
			err := s.putIndex(source, key, idx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/**
* removeData: Removes the model
* @param idx string
* @return error
**/
func (s *Model) removeData(idx string) error {
	data := et.Json{}
	exists, err := s.getObjet(idx, data)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	for _, name := range s.Indexes {
		key := fmt.Sprintf("%v", data[name])
		if key == "" {
			continue
		}

		source := s.stores[name]
		if name == INDEX {
			_, err := source.Delete(key)
			if err != nil {
				return err
			}
			if !s.IsCore {
				return deleteRecord(s.Schema, s.Name, key)
			}
		} else {
			err := s.removeIndex(source, key, idx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/**
* IsDebug: Returns the debug mode
* @return *Model
**/
func (s *Model) IsDebug() *Model {
	s.isDebug = true
	return s
}

/**
* Stricted: Sets the model to strict
* @return void
**/
func (s *Model) Stricted() {
	s.IsStrict = true
}

/**
* Insert: Inserts the model
* @param data et.Json
* @return *Cmd
**/
func (s *Model) Insert(data et.Json) *Cmd {
	result := newCmd(s)
	result.Insert(data)
	return result
}

/**
* update: Updates the model
* @param data et.Json
* @return *Cmd
**/
func (s *Model) Update(data et.Json) *Cmd {
	result := newCmd(s)
	result.Update(data)
	return result
}

/**
* Delete: Deletes the model
* @return *Cmd
**/
func (s *Model) Delete() *Cmd {
	result := newCmd(s)
	result.Delete()
	return result
}

/**
* Upsert: Upserts the model
* @param data et.Json
* @return *Cmd
**/
func (s *Model) Upsert(data et.Json) *Cmd {
	result := newCmd(s)
	result.Upsert(data)
	return result
}

/**
* selects: Returns the select
* @param fields ...string
* @return *Wheres
**/
func (s *Model) Selects(fields ...string) *Wheres {
	result := newWhere()
	result.SetOwner(s)
	for _, field := range fields {
		result.selects = append(result.selects, field)
	}
	return result
}

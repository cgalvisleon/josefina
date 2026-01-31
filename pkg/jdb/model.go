package jdb

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
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
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Host     string `json:"-"`
	IsInit   bool   `json:"-"`
}

/**
* key: Returns the key of the model
* @return string
**/
func (s *From) key() string {
	return modelKey(s.Database, s.Schema, s.Name)
}

/**
* toFrom: Converts a JSON to a From
* @param def et.Json
* @return *From
**/
func toFrom(def et.Json) *From {
	return &From{
		Database: def.Str("database"),
		Schema:   def.Str("schema"),
		Name:     def.Str("name"),
		Host:     def.Str("host"),
		IsInit:   def.Bool("is_init"),
	}
}

type Model struct {
	*From         `json:"from"`
	Fields        map[string]*Field           `json:"fields"`
	Path          string                      `json:"path"`
	Indexes       []string                    `json:"indexes"`
	PrimaryKeys   []string                    `json:"primary_keys"`
	Unique        []string                    `json:"unique"`
	Required      []string                    `json:"required"`
	Hidden        []string                    `json:"hidden"`
	References    map[string]*Detail          `json:"references"`
	Details       map[string]*Detail          `json:"details"`
	Rollups       map[string]*Detail          `json:"rollups"`
	Relations     map[string]*Detail          `json:"relations"`
	Calcs         map[string][]byte           `json:"calcs"`
	BeforeInserts []*Trigger                  `json:"before_inserts"`
	BeforeUpdates []*Trigger                  `json:"before_updates"`
	BeforeDeletes []*Trigger                  `json:"before_deletes"`
	AfterInserts  []*Trigger                  `json:"after_inserts"`
	AfterUpdates  []*Trigger                  `json:"after_updates"`
	AfterDeletes  []*Trigger                  `json:"after_deletes"`
	Version       int                         `json:"version"`
	IsCore        bool                        `json:"is_core"`
	IsStrict      bool                        `json:"is_strict"`
	isDebug       bool                        `json:"-"`
	stores        map[string]*store.FileStore `json:"-"`
	triggers      map[string]*Vm              `json:"-"`
	changed       bool                        `json:"-"`
}

/**
* newModel
* @param database, schema, name string, isCore bool, version int
* @return *Model
**/
func newModel(database, schema, name string, isCore bool, version int) (*Model, error) {
	db, err := getDb(database)
	if err != nil {
		return nil, err
	}
	result, err := db.newModel(schema, name, isCore, version)
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
func (s *Model) ToJson() (et.Json, error) {
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
* Save: Saves the model
* @return error
**/
func (s *Model) Save() error {
	return node.saveModel(s)
}

/**
* store: Opens a store
* @param name string
* @return *store.FileStore, error
**/
func (s *Model) store(name string) (*store.FileStore, error) {
	result, ok := s.stores[name]
	if ok {
		return result, nil
	}

	result, err := store.Open(s.Path, name, s.isDebug)
	if err != nil {
		return nil, err
	}
	s.stores[name] = result

	return result, nil
}

/**
* init: Initializes the model
* @return error
**/
func (s *Model) init() error {
	if s.IsInit {
		return nil
	}

	if len(s.Indexes) == 0 {
		return errors.New(msg.MSG_INDEX_NOT_DEFINED)
	}

	for _, name := range s.Indexes {
		_, err := s.store(name)
		if err != nil {
			return err
		}
	}

	s.IsInit = true
	if node != nil {
		s.Host = node.Host
	}

	for _, detail := range s.Details {
		_, err := getModel(detail.To)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* genKey: Returns a new key for the model
* @return string
**/
func (s *Model) genKey() string {
	return reg.GenUUId(s.Name)
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
* put: Puts the model
* @param idx string, valu any
* @return error
**/
func (s *Model) put(idx string, value any) error {
	source, err := s.source()
	if err != nil {
		return err
	}

	err = source.Put(idx, value)
	if err != nil {
		return err
	}

	return nil
}

/**
* remove: Removes the model
* @param idx string
* @return error
**/
func (s *Model) remove(idx string) error {
	source, err := s.source()
	if err != nil {
		return err
	}

	_, err = source.Delete(idx)
	if err != nil {
		return err
	}

	return nil
}

/**
* get: Gets the model
* @param idx string, dest any
* @return bool, error
**/
func (s *Model) get(idx string, dest any) (bool, error) {
	source, err := s.source()
	if err != nil {
		return false, err
	}

	exists, err := source.Get(idx, &dest)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	return true, nil
}

/**
* putObject: Puts the model
* @param idx string, data et.Json
* @return error
**/
func (s *Model) putObject(idx string, object et.Json) error {
	object[INDEX] = idx
	for _, name := range s.Indexes {
		key := fmt.Sprintf("%v", object[name])
		if key == "" {
			continue
		}

		store := s.stores[name]
		if name == INDEX {
			err := store.Put(key, object)
			if err != nil {
				return err
			}
		} else {
			index := map[string]bool{}
			exists, err := store.Get(key, &index)
			if err != nil {
				return err
			}

			if !exists {
				index = map[string]bool{}
			}

			_, ok := index[idx]
			if ok {
				return nil
			}

			index[idx] = true
			err = store.Put(key, index)
			if err != nil {
				return err
			}

			return nil
		}
	}
	return nil
}

/**
* getObjet: Gets the model as object
* @param idx string
* @return et.Json, error
**/
func (s *Model) getObjet(idx string, dest et.Json) (bool, error) {
	return s.get(idx, &dest)
}

/**
* removeObject: Removes the model
* @param idx string
* @return error
**/
func (s *Model) removeObject(idx string) error {
	data := et.Json{}
	exists, err := s.get(idx, &data)
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

		store := s.stores[name]
		if name == INDEX {
			_, err := store.Delete(key)
			if err != nil {
				return err
			}
		} else {
			index := map[string]bool{}
			exists, err := store.Get(key, &index)
			if err != nil {
				return err
			}

			if !exists {
				return nil
			}

			_, ok := index[idx]
			if !ok {
				return nil
			}

			delete(index, key)
			if len(index) == 0 {
				_, err = store.Delete(key)
				if err != nil {
					return err
				}
				return nil
			}

			err = store.Put(key, index)
			if err != nil {
				return err
			}

			return nil
		}
	}
	return nil
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
func (s *Model) isExisted(field, idx string) (bool, error) {
	source, err := s.store(field)
	if err != nil {
		return false, err
	}

	return source.IsExist(idx), nil
}

/**
* Count: Counts the model
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
* SetDebug
* @param debug bool
**/
func (s *Model) SetDebug(debug bool) {
	s.isDebug = debug
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

/**
* GetModel: Gets the model
* @param from *From
* @return *Model, error
**/
func GetModel(from *From) (*Model, error) {
	if !node.started {
		return nil, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	return node.getModel(from.Database, from.Schema, from.Name)
}

/**
* Put: Puts an object into the model
* @param from *From, key string, data any
* @return error
**/
func Put(from *From, idx string, data any) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.Host != from.Host && from.Host != "" {
		return methods.put(from, idx, data)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.put(idx, data)
}

/**
* Remove: Removes an object from the model
* @param from *From, idx string
* @return error
**/
func Remove(from *From, idx string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.host != from.Host && from.Host != "" {
		return methods.remove(from, idx)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.remove(idx)
}

/**
* Get: Gets an object from the model
* @param from *From, idx string, dest any
* @return bool, error
**/
func Get(from *From, idx string, dest any) (bool, error) {
	if !node.started {
		return false, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.host != from.Host && from.Host != "" {
		return methods.get(from, idx, dest)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return false, fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.get(idx, dest)
}

/**
* PutObject: Puts an object into the model
* @param model *Model, idx string, data et.Json
* @return error
**/
func PutObject(from *From, idx string, data et.Json) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.host != from.Host && from.Host != "" {
		return methods.putObject(from, idx, data)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.putObject(idx, data)
}

/**
* GetObjet: Gets the model as object
* @param idx string
* @return et.Json, error
**/
func GetObjet(from *From, idx string, dest et.Json) (bool, error) {
	return Get(from, idx, &dest)
}

/**
* RemoveObject: Removes an object from the model
* @param model *Model, key string
* @return error
**/
func RemoveObject(from *From, idx string) error {
	if !node.started {
		return errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.host != from.Host && from.Host != "" {
		return methods.removeObject(from, idx)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.removeObject(key)
}

/**
* IsExisted
* @param from *From, field string, key string
* @return (bool, error)
**/
func IsExisted(from *From, field, idx string) (bool, error) {
	if !node.started {
		return false, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.host != from.Host && from.Host != "" {
		return methods.isExisted(from, field, idx)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return false, fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.isExisted(field, idx)
}

/**
* Count
* @param from *From
* @return (int, error)
**/
func Count(from *From) (int, error) {
	if !node.started {
		return 0, errors.New(msg.MSG_NODE_NOT_STARTED)
	}

	if node.host != from.Host && from.Host != "" {
		return methods.count(from)
	}

	key := from.key()
	model, ok := node.models[key]
	if !ok {
		return 0, fmt.Errorf(msg.MSG_MODEL_NOT_FOUND)
	}

	return model.count()
}

package rds

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

type TypeModel string

const (
	KeyValueModel TypeModel = "keyvalue"
	ObjectModel   TypeModel = "object"
	GraphModel    TypeModel = "graph"
)

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
* save: Saves the model
* @return error
**/
func (s *Model) save() error {
	return node.saveModel(s)
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
		fStore, err := store.Open(s.Path, name, s.isDebug)
		if err != nil {
			return err
		}
		s.stores[name] = fStore
	}

	s.IsInit = true
	if node != nil {
		s.Host = node.host
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

	return nil
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
* @param key string
* @return et.Json, error
**/
func (s *Model) getObjet(key string, dest et.Json) (bool, error) {
	return s.get(key, &dest)
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
func (s *Model) isExisted(field, key string) (bool, error) {
	source, err := s.store(field)
	if err != nil {
		return false, err
	}

	return source.IsExist(key), nil
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
* getModel: Gets the model
* @param from *From
* @return *Model, error
**/
func getModel(from *From) (*Model, error) {
	if !node.started {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	return node.getModel(from.Database, from.Schema, from.Name)
}

package rds

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
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
* @param db *DB
* @return error
**/
func initModels(db *DB) error {
	if models != nil {
		return nil
	}

	var err error
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
	Host     string            `json:"host"`
	Fields   map[string]*Field `json:"fields"`
	IsStrict bool              `json:"is_strict"`
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
	isDebug       bool                        `json:"-"`
	db            *DB                         `json:"-"`
	isInit        bool                        `json:"-"`
	data          map[string]*store.FileStore `json:"-"`
	triggers      map[string]*Vm              `json:"-"`
	changed       bool                        `json:"-"`
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
* Key
* @return string
**/
func (s *Model) Key() string {
	result := s.Name
	if s.Schema != "" {
		result = fmt.Sprintf("%s.%s", s.Schema, result)
	}
	if s.Database != "" {
		result = fmt.Sprintf("%s.%s", s.Database, result)
	}
	return result
}

/**
* save: Saves the model
* @return error
**/
func (s *Model) save() error {
	if s.IsCore {
		return nil
	}

	if models == nil {
		return nil
	}

	scr, err := s.serialize()
	if err != nil {
		return err
	}

	key := s.Key()
	err = models.Put(key, scr)
	if err != nil {
		return err
	}

	return nil
}

/**
* prepared: Prepares the model
* @return error
**/
func (s *Model) prepared() error {
	if len(s.Indexes) == 0 {
		return errors.New(msg.MSG_INDEX_NOT_DEFINED)
	}

	return nil
}

/**
* getPath: Returns the path
* @return string, error
**/
func (s *Model) getPath() (string, error) {
	if s.db == nil {
		return "", errors.New(msg.MSG_DB_NOT_FOUND)
	}

	result := strs.Append(s.db.Path, s.Schema, "/")
	result = fmt.Sprintf("%s/%s", result, s.Name)
	return result, nil
}

/**
* init: Initializes the model
* @return error
**/
func (s *Model) init() error {
	if s.isInit {
		return nil
	}

	err := s.prepared()
	if err != nil {
		return err
	}

	path, err := s.getPath()
	if err != nil {
		return err
	}

	for _, name := range s.Indexes {
		fStore, err := store.Open(path, name, s.isDebug)
		if err != nil {
			return err
		}
		s.data[name] = fStore
	}

	s.isInit = true
	return s.save()
}

/**
* index: Returns the index
* @param name string
* @return *store.FileStore, bool
**/
func (s *Model) index(name string) (*store.FileStore, bool) {
	data, ok := s.data[name]
	if !ok {
		return nil, false
	}
	return data, true
}

/**
* count: Counts the model
* @return int
**/
func (s *Model) count() int {
	data, ok := s.index(INDEX)
	if !ok {
		return 0
	}

	return data.Count()
}

/**
* source: Returns the source
* @return *store.FileStore, error
**/
func (s *Model) source() (*store.FileStore, error) {
	result, ok := s.index(INDEX)
	if !ok {
		return nil, errors.New(msg.MSG_INDEX_NOT_FOUND)
	}

	return result, nil
}

/**
* getObjet: Gets the model as object
* @param key string
* @return et.Json, error
**/
func (s *Model) getObjet(key string, dest et.Json) (bool, error) {
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
* getIndex: Gets the index
* @param field, key string, dest map[string]bool
* @return bool, error
**/
func (s *Model) getIndex(field, key string, dest map[string]bool) (bool, error) {
	index, ok := s.index(field)
	if !ok {
		return false, errors.New(msg.MSG_INDEX_NOT_FOUND)
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
	source, ok := s.data[name]
	if !ok {
		return false, errors.New(msg.MSG_INDEX_NOT_FOUND)
	}

	return source.IsExist(key), nil
}

/**
* getKey: Returns a new key for the model
* @return string
**/
func (s *Model) getKey() string {
	return reg.GenUUId(s.Name)
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

		source := s.data[name]
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

		source := s.data[name]
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
* put: Puts the model
* @param idx string, valu any
* @return error
**/
func (s *Model) Put(key string, value any) error {
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
func (s *Model) Remove(key string) error {
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

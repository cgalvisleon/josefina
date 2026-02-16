package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

var (
	ErrorFieldNotFound = errors.New(msg.MSG_FIELD_NOT_FOUND)
)

type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	Address  string `json:"-"`
	isDebug  bool   `json:"-"`
}

/**
* Key: Returns the key of the model
* @return string
**/
func (s *From) Key() string {
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
* ToFrom: Converts a JSON to a From
* @param def et.Json
* @return *From
**/
func ToFrom(def et.Json) *From {
	return &From{
		Database: def.Str("database"),
		Schema:   def.Str("schema"),
		Name:     def.Str("name"),
	}
}

type EventTrigger string

const (
	BeforeInsert EventTrigger = "before_insert"
	AfterInsert  EventTrigger = "after_insert"
	BeforeUpdate EventTrigger = "before_update"
	AfterUpdate  EventTrigger = "after_update"
	BeforeDelete EventTrigger = "before_delete"
	AfterDelete  EventTrigger = "after_delete"
)

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type Model struct {
	*From         `json:"from"`
	IsInit        bool                        `json:"-"`
	Fields        map[string]*Field           `json:"fields"`
	Path          string                      `json:"path"`
	Indexes       []string                    `json:"indexes"`
	PrimaryKeys   []string                    `json:"primary_keys"`
	ForeignKeys   map[string]*Detail          `json:"foreign_keys"`
	Unique        []string                    `json:"unique"`
	Required      []string                    `json:"required"`
	Hidden        []string                    `json:"hidden"`
	Details       map[string]*Detail          `json:"details"`
	Rollups       map[string]*Detail          `json:"rollups"`
	Relations     map[string]*Detail          `json:"relations"`
	Calcs         map[string][]byte           `json:"calcs"`
	BeforeInserts []*Trigger                  `json:"-"`
	AfterInserts  []*Trigger                  `json:"-"`
	BeforeUpdates []*Trigger                  `json:"-"`
	AfterUpdates  []*Trigger                  `json:"-"`
	BeforeDeletes []*Trigger                  `json:"-"`
	AfterDeletes  []*Trigger                  `json:"-"`
	Version       int                         `json:"version"`
	IsCore        bool                        `json:"is_core"`
	IsStrict      bool                        `json:"is_strict"`
	stores        map[string]*store.FileStore `json:"-"`
	schema        *Schema                     `json:"-"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *Model) Serialize() ([]byte, error) {
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
* Init: Initializes the model
* @return error
**/
func (s *Model) Init() error {
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
	return nil
}

/**
* GenKey: Returns a new key for the model
* @return string
**/
func (s *Model) GenKey() string {
	return reg.GenUUId(s.Name)
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
* Source: Returns the source
* @return *store.FileStore, error
**/
func (s *Model) Source() (*store.FileStore, error) {
	result, err := s.store(INDEX)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Put: Puts the model
* @param idx string, value any
* @return error
**/
func (s *Model) Put(idx string, value any) error {
	source, err := s.Source()
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
* Remove: Removes the model
* @param idx string
* @return error
**/
func (s *Model) Remove(idx string) error {
	source, err := s.Source()
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
* Get: Gets the model
* @param idx string, dest any
* @return bool, error
**/
func (s *Model) Get(idx string, dest any) (bool, error) {
	source, err := s.Source()
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
* PutObject: Puts the model
* @param idx string, object et.Json
* @return error
**/
func (s *Model) PutObject(idx string, object et.Json) error {
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
* GetObjet: Gets the model as object
* @param idx string
* @return et.Json, error
**/
func (s *Model) GetObjet(idx string, dest et.Json) (bool, error) {
	return s.Get(idx, &dest)
}

/**
* RemoveObject: Removes the model
* @param idx string
* @return error
**/
func (s *Model) RemoveObject(idx string) error {
	data := et.Json{}
	exists, err := s.Get(idx, &data)
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
* GetIndex: Gets the index
* @param field, key string, dest map[string]bool
* @return bool, error
**/
func (s *Model) GetIndex(field, key string, dest map[string]bool) (bool, error) {
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
* IsExisted: Check if index exists in model
* @param name string, key string
* @return bool, error
**/
func (s *Model) IsExisted(field, idx string) (bool, error) {
	source, err := s.store(field)
	if err != nil {
		return false, err
	}

	return source.IsExist(idx), nil
}

/**
* Exists: Checks if index exists in model
* @param idx string
* @return bool, error
**/
func (s *Model) Exists(idx string) (bool, error) {
	return s.IsExisted(INDEX, idx)
}

/**
* Count: Counts the model
* @return int, error
**/
func (s *Model) Count() (int, error) {
	result, err := s.Source()
	if err != nil {
		return 0, err
	}

	return result.Count(), nil
}

/**
* For: Iterates over the model
* @param fn func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int
* @return bool, error
**/
func (s *Model) For(next func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int) error {
	st, err := s.Source()
	if err != nil {
		return err
	}

	err = st.For(func(idx string, src []byte) (bool, error) {
		item := et.Json{}
		err := json.Unmarshal(src, &item)
		if err != nil {
			return false, err
		}

		return next(idx, item)
	}, asc, offset, limit, workers)
	if err != nil {
		return err
	}

	return nil
}

/**
* AddBeforeInsert
* @param name string, definition []byte
* @return void
**/
func (s *Model) AddBeforeInsert(name string, definition []byte) {
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeInserts[idx].Definition = definition
	} else {
		s.BeforeInserts = append(s.BeforeInserts, &Trigger{Name: name, Definition: definition})
	}
}

/**
* AddAfterInsert
* @param name string, definition []byte
* @return void
**/
func (s *Model) AddAfterInsert(name string, definition []byte) {
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeInserts[idx].Definition = definition
	} else {
		s.BeforeInserts = append(s.BeforeInserts, &Trigger{Name: name, Definition: definition})
	}
}

/**
* AddBeforeUpdate
* @param name string, definition []byte
* @return void
**/
func (s *Model) AddBeforeUpdate(name string, definition []byte) {
	idx := slices.IndexFunc(s.BeforeUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeUpdates[idx].Definition = definition
	} else {
		s.BeforeUpdates = append(s.BeforeUpdates, &Trigger{Name: name, Definition: definition})
	}
}

/**
* AddAfterUpdate
* @param name string, definition []byte
* @return void
**/
func (s *Model) AddAfterUpdate(name string, definition []byte) {
	idx := slices.IndexFunc(s.AfterUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterUpdates[idx].Definition = definition
	} else {
		s.AfterUpdates = append(s.AfterUpdates, &Trigger{Name: name, Definition: definition})
	}
}

/**
* AddBeforeDelete
* @param name string, definition []byte
* @return void
**/
func (s *Model) AddBeforeDelete(name string, definition []byte) {
	idx := slices.IndexFunc(s.BeforeDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeDeletes[idx].Definition = definition
	} else {
		s.BeforeDeletes = append(s.BeforeDeletes, &Trigger{Name: name, Definition: definition})
	}
}

/**
* AddAfterDelete
* @param name string, definition []byte
* @return void
**/
func (s *Model) AddAfterDelete(name string, definition []byte) {
	idx := slices.IndexFunc(s.AfterDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterDeletes[idx].Definition = definition
	} else {
		s.AfterDeletes = append(s.AfterDeletes, &Trigger{Name: name, Definition: definition})
	}
}

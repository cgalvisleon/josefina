package mod

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/josefina/internal/store"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var (
	errorRecordNotFound      = errors.New(msg.MSG_RECORD_NOT_FOUND)
	errorPrimaryKeysNotFound = errors.New(msg.MSG_PRIMARY_KEYS_NOT_FOUND)
	errorFieldNotFound       = errors.New(msg.MSG_FIELD_NOT_FOUND)
	models                   map[string]*Model
)

func init() {
	models = make(map[string]*Model)
}

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type TriggerFunction func(tx *Tx, old, new et.Json) error

type Model struct {
	*From         `json:"from"`
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
	BeforeInserts []*Trigger                  `json:"before_inserts"`
	BeforeUpdates []*Trigger                  `json:"before_updates"`
	BeforeDeletes []*Trigger                  `json:"before_deletes"`
	AfterInserts  []*Trigger                  `json:"after_inserts"`
	AfterUpdates  []*Trigger                  `json:"after_updates"`
	AfterDeletes  []*Trigger                  `json:"after_deletes"`
	Version       int                         `json:"version"`
	IsCore        bool                        `json:"is_core"`
	IsStrict      bool                        `json:"is_strict"`
	stores        map[string]*store.FileStore `json:"-"`
	triggers      map[string]*Vm              `json:"-"`
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

	s.Address = address
	s.IsInit = true
	models[s.Key()] = s
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
* AddBeforeInsert
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeInserts[idx].Definition = fn
	}
	s.BeforeInserts = append(s.BeforeInserts, &Trigger{Name: name, Definition: fn})
}

/**
* AddAfterInsert
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterInserts[idx].Definition = fn
	}
	s.AfterInserts = append(s.AfterInserts, &Trigger{Name: name, Definition: fn})
}

/**
* AddBeforeUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeUpdates[idx].Definition = fn
	}
	s.BeforeUpdates = append(s.BeforeUpdates, &Trigger{Name: name, Definition: fn})
}

/**
* AddAfterUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterUpdates[idx].Definition = fn
	}
	s.AfterUpdates = append(s.AfterUpdates, &Trigger{Name: name, Definition: fn})
}

/**
* AddBeforeDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeDeletes[idx].Definition = fn
	}
	s.BeforeDeletes = append(s.BeforeDeletes, &Trigger{Name: name, Definition: fn})
}

/**
* AddAfterDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterDeletes[idx].Definition = fn
	}
	s.AfterDeletes = append(s.AfterDeletes, &Trigger{Name: name, Definition: fn})
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
* getModel: Returns a model by name
* @param from *From
* @return *Model, error
**/
func getModels(from *From) (*Model, error) {
	key := from.Key()
	result, ok := models[key]
	if ok {
		return result, nil
	}

	return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
}

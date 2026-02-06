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
	errorRecordNotFound      = errors.New(msg.MSG_RECORD_NOT_FOUND)
	errorPrimaryKeysNotFound = errors.New(msg.MSG_PRIMARY_KEYS_NOT_FOUND)
	errorFieldNotFound       = errors.New(msg.MSG_FIELD_NOT_FOUND)
	models                   map[string]*Model
)

func init() {
	models = make(map[string]*Model)
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
	Event      EventTrigger `json:"event"`
	Name       string       `json:"name"`
	Definition []byte       `json:"definition"`
}

type TriggerFunction func(tx *Tx, old, new et.Json) error

type Model struct {
	*From         `json:"from"`
	Address       string                      `json:"-"`
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
	Triggers      []*Trigger                  `json:"triggers"`
	beforeInserts []TriggerFunction           `json:"-"`
	afterInserts  []TriggerFunction           `json:"-"`
	beforeUpdates []TriggerFunction           `json:"-"`
	afterUpdates  []TriggerFunction           `json:"-"`
	beforeDeletes []TriggerFunction           `json:"-"`
	afterDeletes  []TriggerFunction           `json:"-"`
	Version       int                         `json:"version"`
	IsCore        bool                        `json:"is_core"`
	IsStrict      bool                        `json:"is_strict"`
	stores        map[string]*store.FileStore `json:"-"`
	schema        *Schema                     `json:"-"`
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

	s.Address = syn.address
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
* @param fn func(idx string, data et.Json) (bool, error), asc bool, offset, limit, workers int
* @return bool, error
**/
func (s *Model) For(next func(idx string, data et.Json) (bool, error), asc bool, offset, limit, workers int) error {
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
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.Triggers, func(t *Trigger) bool { return t.Name == name && t.Event == BeforeInsert })
	if idx != -1 {
		s.Triggers[idx].Definition = fn
	} else {
		s.Triggers = append(s.Triggers, &Trigger{Event: BeforeInsert, Name: name, Definition: fn})
	}
}

/**
* AddAfterInsert
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.Triggers, func(t *Trigger) bool { return t.Name == name && t.Event == AfterInsert })
	if idx != -1 {
		s.Triggers[idx].Definition = fn
	} else {
		s.Triggers = append(s.Triggers, &Trigger{Event: AfterInsert, Name: name, Definition: fn})
	}
}

/**
* AddBeforeUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.Triggers, func(t *Trigger) bool { return t.Name == name && t.Event == BeforeUpdate })
	if idx != -1 {
		s.Triggers[idx].Definition = fn
	} else {
		s.Triggers = append(s.Triggers, &Trigger{Event: BeforeUpdate, Name: name, Definition: fn})
	}
}

/**
* AddAfterUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.Triggers, func(t *Trigger) bool { return t.Name == name && t.Event == AfterUpdate })
	if idx != -1 {
		s.Triggers[idx].Definition = fn
	} else {
		s.Triggers = append(s.Triggers, &Trigger{Event: AfterUpdate, Name: name, Definition: fn})
	}
}

/**
* AddBeforeDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.Triggers, func(t *Trigger) bool { return t.Name == name && t.Event == BeforeDelete })
	if idx != -1 {
		s.Triggers[idx].Definition = fn
	} else {
		s.Triggers = append(s.Triggers, &Trigger{Event: BeforeDelete, Name: name, Definition: fn})
	}
}

/**
* AddAfterDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.Triggers, func(t *Trigger) bool { return t.Name == name && t.Event == AfterDelete })
	if idx != -1 {
		s.Triggers[idx].Definition = fn
	} else {
		s.Triggers = append(s.Triggers, &Trigger{Event: AfterDelete, Name: name, Definition: fn})
	}
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
* loadModel: Loads a model
* @param model *Model
* @return error
**/
func loadModel(model *Model) (*Model, error) {
	model.IsInit = false
	err := model.Init()
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* GetModel: Returns a model by name
* @param from *From
* @return *Model, bool
**/
func GetModel(from *From) (*Model, bool) {
	key := from.Key()
	result, ok := models[key]
	if ok {
		return result, true
	}

	return nil, false
}

/**
* DropModel: Drops a model
* @param key string
* @return error
**/
func DropModel(key string) error {
	_, ok := models[key]
	if ok {
		delete(models, key)
	}

	return nil
}

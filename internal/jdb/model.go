package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
	"github.com/josefina/internal/msg"
	"github.com/josefina/internal/store"
)

var (
	ErrorFieldNotFound = errors.New(msg.MSG_FIELD_NOT_FOUND)
	ErrorRecordExists  = errors.New(msg.MSG_RECORD_EXISTS)
)

type Trigger struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

type TriggerFnBt func(model *Model, idx string, old []byte, new []byte) error

type Model struct {
	Database      string                      `json:"database"`       // Database name
	Schema        string                      `json:"schema"`         // Schema name
	Name          string                      `json:"name"`           // Model name
	PathData      string                      `json:"path_data"`      // Path to the model
	PathWal       string                      `json:"path_wal"`       // Path to the wal
	Fields        map[string]*Field           `json:"fields"`         // Fields
	Indexes       []*Index                    `json:"indexes"`        // Indexes
	PrimaryKeys   []string                    `json:"primary_keys"`   // Primary keys
	ForeignKeys   map[string]*Detail          `json:"foreign_keys"`   // Foreign keys
	Unique        []*Index                    `json:"unique"`         // Unique
	Required      []*Index                    `json:"required"`       // Required
	Hidden        []string                    `json:"hidden"`         // Hidden
	Details       map[string]*Detail          `json:"details"`        // Details
	Masters       map[string]*Detail          `json:"masters"`        // Masters
	Rollups       map[string]*Detail          `json:"rollups"`        // Rollups
	Relations     map[string]*Detail          `json:"relations"`      // Relations
	Calcs         map[string]string           `json:"calcs"`          // Calculated fields
	BeforeInserts []*Trigger                  `json:"before_inserts"` // Before insert triggers
	AfterInserts  []*Trigger                  `json:"after_inserts"`  // After insert triggers
	BeforeUpdates []*Trigger                  `json:"before_updates"` // Before update triggers
	AfterUpdates  []*Trigger                  `json:"after_updates"`  // After update triggers
	BeforeDeletes []*Trigger                  `json:"before_deletes"` // Before delete triggers
	AfterDeletes  []*Trigger                  `json:"after_deletes"`  // After delete triggers
	Mode          store.Mode                  `json:"mode"`           // Mode
	Version       int                         `json:"version"`        // Version
	IsCore        bool                        `json:"is_core"`        // Is core model
	IsChangue     bool                        `json:"is_changue"`     // Is changue
	IsInit        bool                        `json:"-"`              // Is initialized
	schema        *Schema                     `json:"-"`              // Schema
	db            *DB                         `json:"-"`              // Database
	mu            map[string]*sync.RWMutex    `json:"-"`              // Mutex
	stores        map[string]*store.FileStore `json:"-"`              // Stores
	btrees        map[string]*BTree           `json:"-"`              // Secondary indexes (B+ tree, self-persisting)
	onPut         []TriggerFnBt               `json:"-"`              // On put
	onRemove      []TriggerFnBt               `json:"-"`              // On remove
	isDebug       bool                        `json:"-"`              // Is debug
	isChangue     bool                        `json:"-"`              // Is changue
}

/**
* ToJson: Returns the model as a JSON object
* @return et.Json
**/
func (s *Model) ToJson() et.Json {
	foreignKeys := et.Json{}
	for name, detail := range s.ForeignKeys {
		foreignKeys[name] = detail.ToJson()
	}

	details := et.Json{}
	for name, detail := range s.Details {
		details[name] = detail.ToJson()
	}

	masters := et.Json{}
	for name, detail := range s.Masters {
		masters[name] = detail.ToJson()
	}

	rollups := et.Json{}
	for name, detail := range s.Rollups {
		rollups[name] = detail.ToJson()
	}

	relations := et.Json{}
	for name, detail := range s.Relations {
		relations[name] = detail.ToJson()
	}

	return et.Json{
		"id":           s.Key(),
		"database":     s.Database,
		"schema":       s.Schema,
		"name":         s.Name,
		"foreign_keys": foreignKeys,
		"details":      details,
		"masters":      masters,
		"rollups":      rollups,
		"relations":    relations,
	}
}

/**
* getMutex: Returns the mutex for name
* @param name string
* @return *sync.RWMutex
**/
func (s *Model) getMutex(name string) *sync.RWMutex {
	result, exists := s.mu[name]
	if !exists {
		result = &sync.RWMutex{}
		s.mu[name] = result
	}
	return result
}

/*
* save: Save model data
* @return error
**/
func (s *Model) Save() error {
	if s.IsCore {
		return nil
	}

	if s.db == nil {
		return errors.New(msg.MSG_DB_IS_NIL)
	}

	if s.db.store == nil {
		return errors.New(msg.MSG_STORE_NOT_DEFINED)
	}

	key := s.Key()
	err := s.db.store.put(key, s)
	if err != nil {
		return err
	}

	err = s.db.save()
	if err != nil {
		return err
	}

	return nil
}

/**
* Empty: Empties the model
* @return error
**/
func (s *Model) Empty() error {
	if s.IsCore {
		return nil
	}

	mu := s.getMutex("stores")
	mu.Lock()
	defer mu.Unlock()

	for _, store := range s.stores {
		if err := store.Empty(); err != nil {
			return err
		}
	}

	s.Fields = make(map[string]*Field, 0)
	s.Indexes = make([]*Index, 0)
	s.PrimaryKeys = make([]string, 0)
	s.ForeignKeys = make(map[string]*Detail, 0)
	s.Unique = make([]*Index, 0)
	s.Required = make([]*Index, 0)
	s.Hidden = make([]string, 0)
	s.Details = make(map[string]*Detail, 0)
	s.Masters = make(map[string]*Detail, 0)
	s.Rollups = make(map[string]*Detail, 0)
	s.Relations = make(map[string]*Detail, 0)
	s.Calcs = make(map[string]string, 0)
	s.BeforeInserts = make([]*Trigger, 0)
	s.AfterInserts = make([]*Trigger, 0)
	s.BeforeUpdates = make([]*Trigger, 0)
	s.AfterUpdates = make([]*Trigger, 0)
	s.BeforeDeletes = make([]*Trigger, 0)
	s.AfterDeletes = make([]*Trigger, 0)
	s.btrees = make(map[string]*BTree, 0)

	return nil
}

/**
* Key: Get the key of the model
* @return string
**/
func (s *Model) Key() string {
	result := s.Database
	result = strs.Append(result, s.Schema, ".")
	return strs.Append(result, s.Name, ".")
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
* GenKey: Returns a new key for the model
* @return string
**/
func (s *Model) GenKey() string {
	return reg.GenULID(s.Name)
}

/**
* Init: Opens the primary store and all secondary BTree indexes.
* Each BTree loads its own persisted data on open.
* @return error
**/
func (s *Model) Init() error {
	if s.IsInit {
		return nil
	}

	if len(s.Indexes) == 0 {
		return errors.New(msg.MSG_INDEX_NOT_DEFINED)
	}

	// Open each secondary BTree (Init loads persisted data from its own store).
	for _, index := range s.Indexes {
		if strings.EqualFold(INDEX, index.Name) {
			if err := s.loadStore(index.Name); err != nil {
				return err
			}
			continue
		}
		if index.Type == TpIndexBTree {
			if err := s.loadBTree(index.Name); err != nil {
				return err
			}
		}
	}

	s.IsInit = true
	return nil
}

/**
* OnPut: Adds a function to be called when a document is put
* @param fn TriggerFnBt
**/
func (s *Model) OnPut(fn TriggerFnBt) {
	s.onPut = append(s.onPut, fn)
}

/**
* OnRemove: Adds a function to be called when a document is removed
* @param fn TriggerFnBt
**/
func (s *Model) OnRemove(fn TriggerFnBt) {
	s.onRemove = append(s.onRemove, fn)
}

/**
* getStore: Returns the store for name
* @param name string
* @return *store.FileStore, bool
**/
func (s *Model) getStore(name string) (*store.FileStore, bool) {
	mu := s.getMutex("stores")
	mu.RLock()
	result, exists := s.stores[name]
	mu.RUnlock()

	return result, exists
}

/**
* loadStore: Loads a store
* @param name string
* @return error
**/
func (s *Model) loadStore(name string) error {
	result, exists := s.getStore(name)
	if exists {
		return nil
	}

	result, err := store.Open(s.PathData, s.PathWal, name, s.Mode)
	if err != nil {
		return err
	}

	mu := s.getMutex("stores")
	mu.Lock()
	s.stores[name] = result
	mu.Unlock()

	return nil
}

/**
* source: Returns the primary store
* @return *store.FileStore, error
**/
func (s *Model) source() (*store.FileStore, bool) {
	return s.getStore(INDEX)
}

/**
* getBtree returns the BTree for field, checking if it exists.
* @param field string
* @return *BTree, bool
**/
func (s *Model) getBtree(field string) (*BTree, bool) {
	mu := s.getMutex("btrees")
	mu.RLock()
	bt, exists := s.btrees[field]
	mu.RUnlock()
	if exists {
		return bt, true
	}

	return nil, false
}

/**
* loadBTree returns the BTree for field, opening and loading it on first access.
* @param field string
* @return error
**/
func (s *Model) loadBTree(field string) error {
	bt, exists := s.getBtree(field)
	if exists {
		return nil
	}

	bt, err := OpenBTree(s.PathData, s.PathWal, field)
	if err != nil {
		return err
	}

	mu := s.getMutex("btrees")
	mu.Lock()
	s.btrees[field] = bt
	mu.Unlock()
	return nil
}

/**
* fireTriggers: Fires the triggers
* @param triggers []*Trigger, old, new et.Json
* @return error
**/
func (s *Model) fireTriggers(trigger *Trigger, old, new *et.Json) error {
	v := jrex.NewInstance()
	v.SetCode(trigger.Definition)
	v.Set("Old", old)
	v.Set("New", new)
	_, err := v.Run()
	if err != nil {
		return err
	}

	return nil
}

/**
* put: Puts a raw value by primary key
* @param idx string, value any, expiration time.Duration
* @return error
**/
func (s *Model) put(idx string, value any) error {
	store, exists := s.source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	_, old, err := store.Get(idx)
	if err != nil {
		return err
	}

	bt, _, err := store.Put(idx, value)
	if err != nil {
		return err
	}

	// Call onPut functions
	for _, fn := range s.onPut {
		if err := fn(s, idx, bt, old); err != nil {
			return err
		}
	}

	return nil
}

/**
* remove: Removes a document by primary key (no index cleanup)
* @param idx string
* @return error
**/
func (s *Model) remove(idx string) error {
	store, exists := s.source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	exists, old, err := store.Get(idx)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	_, err = store.Delete(idx)
	if err != nil {
		return err
	}

	// Call onRemove functions
	for _, fn := range s.onRemove {
		if err := fn(s, idx, old, nil); err != nil {
			return err
		}
	}

	return err
}

/**
* Get: Gets a document by primary key
* @param idx string, dest any
* @return bool, error
**/
func (s *Model) get(idx string, dest any) (bool, error) {
	store, exists := s.source()
	if !exists {
		return false, errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	exists, data, err := store.Get(idx)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

/**
* putObject: Inserts or updates a document and keeps secondary indexes in sync.
* @param idx string, object et.Json
* @return error
**/
func (s *Model) putObject(idx string, object et.Json) error {
	object[INDEX] = idx
	err := s.put(idx, object)
	if err != nil {
		return err
	}

	// Update each secondary BTree index.
	for _, index := range s.Indexes {
		if index.Name == INDEX {
			continue
		}
		v := object[index.Name]
		if v == nil {
			continue
		}
		if index.Type == TpIndexBTree {
			bt, exists := s.getBtree(index.Name)
			if !exists {
				continue
			}
			key := KeyFromAny(v)
			if err := bt.Insert(key, idx); err != nil {
				return err
			}
		}
	}

	return nil
}

/**
* deleteObject: Removes a document and cleans up all secondary indexes.
* @param idx string
* @return error
**/
func (s *Model) deleteObject(idx string) error {
	current, exists, err := s.current(idx)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	// Remove from each secondary BTree index.
	for _, index := range s.Indexes {
		if index.Name == INDEX {
			continue
		}
		v := current[index.Name]
		if v == nil {
			continue
		}
		if index.Type == TpIndexBTree {
			bt, exists := s.getBtree(index.Name)
			if !exists {
				continue
			}
			key := KeyFromAny(v)
			if _, err := bt.Delete(key, idx); err != nil {
				return err
			}
		}
	}

	err = s.delete(idx)
	if err != nil {
		return err
	}

	return nil
}

/**
* getObject: Gets an object by primary key
* @param idx string
* @return et.Json, error
**/
func (s *Model) getObject(idx string) (et.Json, error) {
	var dest et.Json
	exists, err := s.get(idx, &dest)
	if err != nil {
		return et.Json{}, err
	}

	if !exists {
		return et.Json{}, errors.New(msg.MSG_RECORD_NOT_FOUND)
	}

	return dest, nil
}

/**
* current: Gets the current document by primary key
* @param idx string
* @return et.Json, bool, error
**/
func (s *Model) current(idx string) (et.Json, bool, error) {
	var dest et.Json
	exists, err := s.get(idx, &dest)
	if err != nil {
		return et.Json{}, false, err
	}
	return dest, exists, nil
}

/**
* isExists: Checks if a document exists by primary key
* @param idx string
* @return bool, error
**/
func (s *Model) isExists(idx string) (bool, error) {
	source, exists := s.source()
	if !exists {
		return false, errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	return source.IsExist(idx), nil
}

/**
* Count: Counts documents in the primary store
* @return int, error
**/
func (s *Model) count() (int, error) {
	result, exists := s.source()
	if !exists {
		return 0, errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	return result.Count(), nil
}

/**
* insert: Inserts a document and keeps secondary indexes in sync.
* @param idx string, data et.Json
* @return error
**/
func (s *Model) insert(idx string, new et.Json) error {
	exists, err := s.isExists(idx)
	if err != nil {
		return err
	}

	if exists {
		return ErrorRecordExists
	}

	for _, index := range s.Required {
		_, exists := new[index.Name]
		if !exists {
			return fmt.Errorf(msg.MSG_REQUIRED_FIELD, index.Tag)
		}
	}

	for _, index := range s.Unique {
		value := new[index.Name]
		if value == nil {
			continue
		}

		if index.Type == TpIndexBTree {
			btree, exists := s.getBtree(index.Name)
			if exists {
				key := KeyFromAny(value)
				_, ok := btree.Get(key)
				if ok {
					return fmt.Errorf(msg.MSG_DUPLICATE_KEY_UNIQUE, index.Tag)
				}
			}
		}
	}

	var old = et.Json{}
	for _, trigger := range s.BeforeInserts {
		err := s.fireTriggers(trigger, &old, &new)
		if err != nil {
			return err
		}
	}

	err = s.putObject(idx, new)
	if err != nil {
		return err
	}

	for _, trigger := range s.AfterInserts {
		err = s.fireTriggers(trigger, &old, &new)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* update: Updates a document and keeps secondary indexes in sync.
* @param idx string, new et.Json
* @return error
**/
func (s *Model) update(idx string, new et.Json) error {
	if idx == "" {
		return errors.New(msg.MSG_RECORD_NOT_FOUND)
	}

	old, exists, err := s.current(idx)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New(msg.MSG_RECORD_NOT_FOUND)
	}

	for _, index := range s.Required {
		_, exists := new[index.Name]
		if !exists {
			return fmt.Errorf(msg.MSG_REQUIRED_FIELD, index.Tag)
		}
	}

	for _, index := range s.Unique {
		value := new[index.Name]
		if value == nil {
			continue
		}

		if index.Type == TpIndexBTree {
			btree, exists := s.getBtree(index.Name)
			if exists {
				key := KeyFromAny(value)
				_, ok := btree.Get(key)
				if ok {
					return fmt.Errorf(msg.MSG_DUPLICATE_KEY_UNIQUE, index.Tag)
				}
			}
		}
	}

	for _, trigger := range s.BeforeUpdates {
		err := s.fireTriggers(trigger, &old, &new)
		if err != nil {
			return err
		}
	}

	err = s.putObject(idx, new)
	if err != nil {
		return err
	}

	for _, trigger := range s.AfterUpdates {
		err := s.fireTriggers(trigger, &old, &new)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* delete: Removes a document and cleans up all secondary indexes.
* @param idx string
* @return error
**/
func (s *Model) delete(idx string) error {
	if idx == "" {
		return errors.New(msg.MSG_RECORD_NOT_FOUND)
	}

	old, exists, err := s.current(idx)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	new := et.Json{}
	for _, trigger := range s.BeforeDeletes {
		err := s.fireTriggers(trigger, &old, &new)
		if err != nil {
			return err
		}
	}

	err = s.remove(idx)
	if err != nil {
		return err
	}

	for _, trigger := range s.AfterDeletes {
		err := s.fireTriggers(trigger, &old, &new)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* upsert: Inserts or updates a document and keeps secondary indexes in sync.
* @param new et.Json, tx *Tx
* @return error
**/
func (s *Model) upsert(idx string, new et.Json) error {
	err := s.insert(idx, new)
	if errors.Is(err, ErrorRecordExists) {
		return s.update(idx, new)
	} else if err != nil {
		return err
	}

	return nil
}

/**
* AddBeforeInsert
* @param name string, definition string
**/
func (s *Model) AddBeforeInsert(name string, definition string) {
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeInserts[idx].Definition = definition
	} else {
		s.BeforeInserts = append(s.BeforeInserts, &Trigger{
			Name:       name,
			Definition: definition,
		})
	}
}

/**
* AddAfterInsert
* @param name string, definition string
**/
func (s *Model) AddAfterInsert(name string, definition string) {
	idx := slices.IndexFunc(s.AfterInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterInserts[idx].Definition = definition
	} else {
		s.AfterInserts = append(s.AfterInserts, &Trigger{
			Name:       name,
			Definition: definition,
		})
	}
}

/**
* AddBeforeUpdate
* @param name string, definition string
**/
func (s *Model) AddBeforeUpdate(name string, definition string) {
	idx := slices.IndexFunc(s.BeforeUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeUpdates[idx].Definition = definition
	} else {
		s.BeforeUpdates = append(s.BeforeUpdates, &Trigger{
			Name:       name,
			Definition: definition,
		})
	}
}

/**
* AddAfterUpdate
* @param name string, definition string
**/
func (s *Model) AddAfterUpdate(name string, definition string) {
	idx := slices.IndexFunc(s.AfterUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterUpdates[idx].Definition = definition
	} else {
		s.AfterUpdates = append(s.AfterUpdates, &Trigger{
			Name:       name,
			Definition: definition,
		})
	}
}

/**
* AddBeforeDelete
* @param name string, definition string
**/
func (s *Model) AddBeforeDelete(name string, definition string) {
	idx := slices.IndexFunc(s.BeforeDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeDeletes[idx].Definition = definition
	} else {
		s.BeforeDeletes = append(s.BeforeDeletes, &Trigger{
			Name:       name,
			Definition: definition,
		})
	}
}

/**
* AddAfterDelete
* @param name string, definition string
**/
func (s *Model) AddAfterDelete(name string, definition string) {
	idx := slices.IndexFunc(s.AfterDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterDeletes[idx].Definition = definition
	} else {
		s.AfterDeletes = append(s.AfterDeletes, &Trigger{
			Name:       name,
			Definition: definition,
		})
	}
}

/**
* ForEachOfBytes: Iterates over all documents in the primary store
* @param next func(idx string, src []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Model) ForEachOfBytes(next func(idx string, src []byte) (bool, error), asc bool, offset, limit int) error {
	st, exists := s.source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	return st.ForEach(func(idx string, src []byte) (bool, error) {
		return next(idx, src)
	}, asc, offset, limit)
}

/**
* ForEach: Iterates over all documents in the primary store
* @param next func(idx string, item et.Json) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Model) ForEach(next func(idx string, item et.Json) (bool, error), asc bool, offset, limit int) error {
	st, exists := s.source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	return st.ForEach(func(idx string, src []byte) (bool, error) {
		item := et.Json{}
		if err := json.Unmarshal(src, &item); err != nil {
			return false, err
		}
		return next(idx, item)
	}, asc, offset, limit)
}

/**
* CreateIndex: Creates an index for the specified field
* @param name string
* @return error
**/
func (s *Model) CreateIndex(name string, tp TpIndex) error {
	_, exists := s.findIndex(name)
	if exists {
		return nil
	}

	_, err := s.DefineIndex(name, tp)
	if err != nil {
		return err
	}

	if tp != TpIndexBTree {
		return nil
	}

	bt, exists := s.getBtree(name)
	if !exists {
		return nil
	}

	return s.ForEach(func(idx string, item et.Json) (bool, error) {
		v := item[name]
		if v == nil {
			return true, nil
		}

		key := KeyFromAny(v)
		if err := bt.Insert(key, idx); err != nil {
			return true, err
		}
		return true, nil
	}, true, 0, 0)
}

/**
* Insert: Inserts a document
* @param item et.Json
* @return *Command
**/
func (s *Model) Insert(item et.Json) *Command {
	return newCommand(s, INSERT, []et.Json{item})
}

/**
* Update: Updates a document
* @param item et.Json
* @return *Command
**/
func (s *Model) Update(item et.Json) *Command {
	return newCommand(s, UPDATE, []et.Json{item})
}

/**
* Delete: Deletes a document
* @return *Command
**/
func (s *Model) Delete() *Command {
	return newCommand(s, DELETE, []et.Json{})
}

/**
* Upsert: Inserts or updates a document
* @param item et.Json
* @return *Command
**/
func (s *Model) Upsert(item et.Json) *Command {
	return newCommand(s, UPSERT, []et.Json{item})
}

/**
* Exec: Executes the command
* @return et.Items, error
**/
func (s *Model) Bulk(items []et.Json) *Command {
	return newCommand(s, BULK, items)
}

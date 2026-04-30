package jdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

var (
	ErrorFieldNotFound = errors.New(msg.MSG_FIELD_NOT_FOUND)
)

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type Model struct {
	Database      string                                   `json:"database"`       // Database name
	Schema        string                                   `json:"schema"`         // Schema name
	Name          string                                   `json:"name"`           // Model name
	IsInit        bool                                     `json:"-"`              // Is initialized
	Path          string                                   `json:"path"`           // Path to the model
	Fields        map[string]*Field                        `json:"fields"`         // Fields
	Indexes       []*Index                                 `json:"indexes"`        // Indexes
	PrimaryKeys   []string                                 `json:"primary_keys"`   // Primary keys
	ForeignKeys   map[string]*Detail                       `json:"foreign_keys"`   // Foreign keys
	Unique        []*Index                                 `json:"unique"`         // Unique
	Required      []string                                 `json:"required"`       // Required
	Hidden        []string                                 `json:"hidden"`         // Hidden
	Details       map[string]*Detail                       `json:"details"`        // Details
	Rollups       map[string]*Detail                       `json:"rollups"`        // Rollups
	Relations     map[string]*Detail                       `json:"relations"`      // Relations
	Calcs         map[string][]byte                        `json:"calcs"`          // Calculated fields
	BeforeInserts []*Trigger                               `json:"before_inserts"` // Before insert triggers
	AfterInserts  []*Trigger                               `json:"after_inserts"`  // After insert triggers
	BeforeUpdates []*Trigger                               `json:"before_updates"` // Before update triggers
	AfterUpdates  []*Trigger                               `json:"after_updates"`  // After update triggers
	BeforeDeletes []*Trigger                               `json:"before_deletes"` // Before delete triggers
	AfterDeletes  []*Trigger                               `json:"after_deletes"`  // After delete triggers
	Mode          store.Mode                               `json:"mode"`           // Mode
	Version       int                                      `json:"version"`        // Version
	IsCore        bool                                     `json:"is_core"`        // Is core model
	IsChangue     bool                                     `json:"is_changue"`     // Is changue
	IsStrict      bool                                     `json:"is_strict"`      // Is strict model
	stores        map[string]*store.FileStore              `json:"-"`              // Stores
	btrees        map[string]*BTree                        `json:"-"`              // Secondary indexes (B+ tree, self-persisting)
	schema        *Schema                                  `json:"-"`              // Schema
	db            *DB                                      `json:"-"`              // Database
	node          *Node                                    `json:"-"`              // Node
	mu            *sync.RWMutex                            `json:"-"`              // Mutex
	onPut         []func(*Node, string, any, string) error `json:"-"`              // On put
	onRemove      []func(*Node, string, any) error         `json:"-"`              // On remove
	isDebug       bool                                     `json:"-"`              // Is debug
}

/**
* newModel: Creates a new model
* @param *Schema s, string name, string path, int version, bool isCore
* @return (*Model, error)
**/
func newModel(s *Schema, name, path string, version int, isCore bool) (*Model, error) {
	result := &Model{
		Database:      s.Database,
		Schema:        s.Name,
		Name:          name,
		Path:          path,
		Fields:        make(map[string]*Field, 0),
		Indexes:       make([]*Index, 0),
		PrimaryKeys:   make([]string, 0),
		ForeignKeys:   make(map[string]*Detail, 0),
		Unique:        make([]*Index, 0),
		Required:      make([]string, 0),
		Hidden:        make([]string, 0),
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
		Mode:          store.ReadWrite,
		Version:       version,
		IsCore:        isCore,
		IsChangue:     false,
		stores:        make(map[string]*store.FileStore, 0),
		btrees:        make(map[string]*BTree, 0),
		schema:        s,
		db:            s.db,
		node:          s.db.node,
		mu:            &sync.RWMutex{},
		onPut:         make([]func(*Node, string, any, string) error, 0),
		onRemove:      make([]func(*Node, string, any) error, 0),
	}
	_, err := result.defineSource()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* load: Load model data
* @param *Schema schema
* @return error
**/
func (s *Model) load(schema *Schema) error {
	s.stores = make(map[string]*store.FileStore, 0)
	s.btrees = make(map[string]*BTree, 0)
	s.schema = schema
	s.db = schema.db
	s.node = schema.db.node
	s.onPut = make([]func(*Node, string, any, string) error, 0)
	s.onRemove = make([]func(*Node, string, any) error, 0)

	if err := s.Init(); err != nil {
		return err
	}

	return nil
}

/*
* save: Save model data
* @return error
**/
func (s *Model) Save() error {
	if s.IsCore {
		return nil
	}
	return s.schema.save()
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
* Serialize
* @return []byte, error
*
 */
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
* IsDebug: Returns the debug mode
* @return *Model
**/
func (s *Model) IsDebug() *Model {
	s.isDebug = true
	return s
}

/**
* Stricted: Sets the model to strict
**/
func (s *Model) Stricted() {
	s.IsStrict = true
}

/**
* GenKey: Returns a new key for the model
* @return string
**/
func (s *Model) GenKey() string {
	return reg.GenUUId(s.Name)
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
* getBtree returns the BTree for field, checking if it exists.
* @param field string
* @return *BTree, bool
**/
func (s *Model) getBtree(field string) (*BTree, bool) {
	s.mu.RLock()
	bt, exists := s.btrees[field]
	s.mu.RUnlock()
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

	bt, err := OpenBTree(s.Path, field)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.btrees[field] = bt
	s.mu.Unlock()
	return nil
}

/**
* OnPut: Adds a function to be called when a document is put
* @param fn func(string, any, string) error
**/
func (s *Model) OnPut(fn func(*Node, string, any, string) error) {
	s.onPut = append(s.onPut, fn)
}

/**
* OnRemove: Adds a function to be called when a document is removed
* @param fn func(string, any) error
**/
func (s *Model) OnRemove(fn func(*Node, string, any) error) {
	s.onRemove = append(s.onRemove, fn)
}

/**
* Store: Returns the store for name
* @param name string
* @return *store.FileStore, bool
**/
func (s *Model) GetStore(name string) (*store.FileStore, bool) {
	s.mu.RLock()
	result, exists := s.stores[name]
	s.mu.RUnlock()

	return result, exists
}

/**
* loadStore: Loads a store
* @param name string
* @return error
**/
func (s *Model) loadStore(name string) error {
	result, exists := s.GetStore(name)
	if exists {
		return nil
	}

	result, err := store.Open(s.Path, name, s.Mode)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.stores[name] = result
	s.mu.Unlock()

	return nil
}

/**
* Source: Returns the primary store
* @return *store.FileStore, error
**/
func (s *Model) Source() (*store.FileStore, bool) {
	return s.GetStore(INDEX)
}

/**
* Put: Puts a raw value by primary key
* @param idx string, value any, expiration time.Duration
* @return error
**/
func (s *Model) Put(idx string, value any) error {
	store, exists := s.Source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	exists, err := store.Put(idx, value)
	if err != nil {
		return err
	}

	// Call onPut functions
	for _, fn := range s.onPut {
		command := INSERT
		if exists {
			command = UPDATE
		}
		if err := fn(s.node, idx, value, command); err != nil {
			return err
		}
	}

	return nil
}

/**
* Get: Gets a document by primary key
* @param idx string, dest any
* @return bool, error
**/
func (s *Model) Get(idx string, dest any) (bool, error) {
	store, exists := s.Source()
	if !exists {
		return false, errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	exists, err := store.Get(idx, &dest)
	if err != nil {
		return false, err
	}

	return exists, nil
}

/**
* remove: Removes a document by primary key (no index cleanup)
* @param idx string, val any
* @return error
**/
func (s *Model) remove(idx string, val any) error {
	store, exists := s.Source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	exists, err := store.Delete(idx)
	if exists {
		// Call onRemove functions
		for _, fn := range s.onRemove {
			if err := fn(s.node, idx, val); err != nil {
				return err
			}
		}
	}

	return err
}

/**
* Remove: Removes a document by primary key (no index cleanup)
* @param idx string
* @return error
**/
func (s *Model) Remove(idx string) error {
	return s.remove(idx, nil)
}

/**
* putObject: Inserts or updates a document and keeps secondary indexes in sync.
* @param idx string, object et.Json
* @return error
**/
func (s *Model) putObject(idx string, object et.Json) error {
	object[INDEX] = idx
	err := s.Put(idx, object)
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
			if err := bt.Insert(KeyFromAny(v), idx); err != nil {
				return err
			}
		}
	}

	return nil
}

/**
* deleteObject: Removes a document and cleans up all secondary indexes.
* @param idx string, current et.Json
* @return error
**/
func (s *Model) deleteObject(idx string, current et.Json) error {
	store, exists := s.Source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	exists = store.IsExist(idx)
	if !exists {
		return nil
	}

	err := s.remove(idx, current)
	if err != nil {
		return err
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
			if _, err := bt.Delete(KeyFromAny(v), idx); err != nil {
				return err
			}
		}
	}

	return nil
}

/**
* Current: Gets the current document by primary key
* @param idx string
* @return et.Json, bool, error
**/
func (s *Model) Current(idx string) (et.Json, bool, error) {
	var dest et.Json
	exists, err := s.Get(idx, &dest)
	if err != nil {
		return et.Json{}, false, err
	}
	return dest, exists, nil
}

/**
* IsExists: Checks if a document exists by primary key
* @param idx string
* @return bool, error
**/
func (s *Model) IsExists(idx string) (bool, error) {
	source, exists := s.Source()
	if !exists {
		return false, errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	return source.IsExist(idx), nil
}

/**
* Count: Counts documents in the primary store
* @return int, error
**/
func (s *Model) Count() (int, error) {
	result, exists := s.Source()
	if !exists {
		return 0, errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	return result.Count(), nil
}

/**
* fireTriggers: Fires the triggers
* @param triggers []*Trigger, old, new et.Json
* @return error
**/
func (s *Model) fireTriggers(trigger *Trigger, old, new *et.Json, tx *Tx) (*Tx, error) {
	name := fmt.Sprintf("trigger_%s", trigger.Name)
	vm, err := js.New(name)
	if err != nil {
		return tx, err
	}

	vm.Set("Old", old)
	vm.Set("New", new)
	vm.Set("Tx", tx)
	_, err = vm.Run(string(trigger.Definition))
	if err != nil {
		return tx, err
	}

	return tx, nil
}

/**
* insert: Inserts a document and keeps secondary indexes in sync.
* @param idx string, data et.Json, tx *Tx
* @return (*Transaction, error)
**/
func (s *Model) insert(idx string, data et.Json, tx *Tx) (*Tx, error) {
	tx = GetTx(s.schema.db, tx)
	new := et.Json{}
	if s.IsStrict {
		for _, field := range s.Fields {
			if value, ok := data[field.Name]; ok {
				new[field.Name] = value
			}
		}
	} else {
		new = data
	}

	cache := tx.Items(s)
	for _, index := range s.Unique {
		value := new[index.Name]
		if value != nil {
			if index.Type == TpIndexBTree {
				btree, exists := s.getBtree(index.Name)
				if exists {
					_, ok := btree.Get(KeyFromAny(value))
					if ok {
						return tx, fmt.Errorf(msg.MSG_DUPLICATE_KEY_UNIQUE, index.Tag)
					}
				}
			}
		}
		result := et.From(cache).
			Where(et.Eq(index.Name, value)).
			All()
		if len(result) > 0 {
			return tx, fmt.Errorf(msg.MSG_DUPLICATE_KEY_UNIQUE, index.Tag)
		}
	}

	var old = et.Json{}
	for _, trigger := range s.BeforeInserts {
		tx, err := s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	tx, err := tx.Add(s, INSERT, idx, old, new)
	if err != nil {
		return tx, err
	}

	for _, trigger := range s.AfterInserts {
		tx, err := s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	return tx, nil
}

/**
* Insert: Inserts a document and keeps secondary indexes in sync.
* @param idx string, data et.Json, tx *Tx
* @return (*Transaction, error)
**/
func (s *Model) Insert(idx string, data et.Json, tx *Tx) (*Tx, error) {
	exists, err := s.IsExists(idx)
	if err != nil {
		return tx, err
	}
	if exists {
		return tx, errors.New(msg.MSG_RECORD_EXISTS)
	}

	return s.insert(idx, data, tx)
}

/**
* Update: Updates a document and keeps secondary indexes in sync.
* @param idx string, data et.Json
* @return (et.Json, error)
**/
func (s *Model) Update(idx string, data et.Json, tx *Tx) (*Tx, error) {
	tx = GetTx(s.schema.db, tx)
	new := et.Json{}
	if s.IsStrict {
		for _, field := range s.Fields {
			if value, ok := data[field.Name]; ok {
				new[field.Name] = value
			}
		}
	} else {
		new = data
	}

	var old et.Json
	exists, err := s.Get(idx, &old)
	if err != nil {
		return tx, err
	}
	if !exists {
		return tx, fmt.Errorf(msg.MSG_RECORD_NOT_FOUND)
	}

	cache := tx.Items(s)
	for _, index := range s.Unique {
		value := new[index.Name]
		if value != nil {
			if index.Type == TpIndexBTree {
				btree, exists := s.getBtree(index.Name)
				if exists {
					idxs, ok := btree.Get(KeyFromAny(value))
					if ok {
						i := slices.Index(idxs, idx)
						if i == -1 {
							return tx, fmt.Errorf(msg.MSG_DUPLICATE_KEY_UNIQUE, index.Tag)
						}
					}
				}
			}
		}
		result := et.From(cache).
			Where(et.Eq(index.Name, value)).
			And(et.Neg(INDEX, idx)).
			All()
		if len(result) > 0 {
			return tx, fmt.Errorf(msg.MSG_DUPLICATE_KEY_UNIQUE, index.Tag)
		}
	}

	for _, trigger := range s.BeforeUpdates {
		tx, err := s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	tx.Add(s, UPDATE, idx, old, new)

	for _, trigger := range s.AfterUpdates {
		tx, err := s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	return tx, nil
}

/**
* Upsert: Inserts or updates a document and keeps secondary indexes in sync.
* @param idx string, new et.Json, tx *Tx
* @return (*Transaction, error)
**/
func (s *Model) Upsert(idx string, new et.Json, tx *Tx) (*Tx, error) {
	exists, err := s.IsExists(idx)
	if err != nil {
		return tx, err
	}
	if exists {
		return s.Update(idx, new, tx)
	}

	return s.insert(idx, new, tx)
}

/**
* Delete: Removes a document and cleans up all secondary indexes.
* @param idx string
* @return error
**/
func (s *Model) Delete(idx string, tx *Tx) (*Tx, error) {
	tx = GetTx(s.schema.db, tx)
	var old et.Json
	exists, err := s.Get(idx, &old)
	if err != nil {
		return tx, err
	}
	if !exists {
		return tx, nil
	}

	new := et.Json{}
	for _, trigger := range s.BeforeDeletes {
		tx, err := s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	tx, err = tx.Add(s, DELETE, idx, old, new)
	if err != nil {
		return tx, err
	}

	for _, trigger := range s.AfterDeletes {
		tx, err = s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	return tx, nil
}

/**
* Value: Returns the value of the specified attribute from the given item.
* @param atrib string, item et.Json
* @return (interface{}, bool)
**/
func (s *Model) Value(atrib string, item et.Json) (interface{}, bool) {
	field, exists := s.Fields[atrib]
	if exists {
		return field.Value(item), true
	}

	val, ok := item[atrib]
	if ok {
		return val, true
	}

	return nil, false
}

/**
* ForEach: Iterates over all documents in the primary store
* @param next func(idx string, src []byte) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Model) ForEachBt(next func(idx string, src []byte) (bool, error), asc bool, offset, limit int) error {
	st, exists := s.Source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	total := st.Count()
	workers := total / 1000
	if workers <= 0 {
		workers = 1
	}

	return st.ForEach(func(idx string, src []byte) (bool, error) {
		return next(idx, src)
	}, asc, offset, limit, workers)
}

/**
* ForEach: Iterates over all documents in the primary store
* @param next func(idx string, item et.Json) (bool, error), asc bool, offset, limit int
* @return error
**/
func (s *Model) ForEach(next func(idx string, item et.Json) (bool, error), asc bool, offset, limit int) error {
	st, exists := s.Source()
	if !exists {
		return errors.New(msg.MSG_STORE_NOT_FOUND)
	}

	total := st.Count()
	workers := total / 1000
	if workers <= 0 {
		workers = 1
	}

	return st.ForEach(func(idx string, src []byte) (bool, error) {
		item := et.Json{}
		if err := json.Unmarshal(src, &item); err != nil {
			return false, err
		}
		return next(idx, item)
	}, asc, offset, limit, workers)
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

	return s.ForEach(func(idx string, item et.Json) (bool, error) {
		v := item[name]
		if v == nil {
			return true, nil
		}
		if tp != TpIndexBTree {
			return true, nil
		}
		bt, exists := s.getBtree(name)
		if !exists {
			return true, nil
		}
		if err := bt.Insert(KeyFromAny(v), idx); err != nil {
			return true, err
		}
		return true, nil
	}, true, 0, 0)
}

/**
* AddBeforeInsert
* @param name string, definition string
**/
func (s *Model) AddBeforeInsert(name string, definition string) {
	bt := []byte(definition)
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeInserts[idx].Definition = bt
	} else {
		s.BeforeInserts = append(s.BeforeInserts, &Trigger{Name: name, Definition: bt})
	}
}

/**
* AddAfterInsert
* @param name string, definition string
**/
func (s *Model) AddAfterInsert(name string, definition string) {
	bt := []byte(definition)
	idx := slices.IndexFunc(s.AfterInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterInserts[idx].Definition = bt
	} else {
		s.AfterInserts = append(s.AfterInserts, &Trigger{Name: name, Definition: bt})
	}
}

/**
* AddBeforeUpdate
* @param name string, definition string
**/
func (s *Model) AddBeforeUpdate(name string, definition string) {
	bt := []byte(definition)
	idx := slices.IndexFunc(s.BeforeUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeUpdates[idx].Definition = bt
	} else {
		s.BeforeUpdates = append(s.BeforeUpdates, &Trigger{Name: name, Definition: bt})
	}
}

/**
* AddAfterUpdate
* @param name string, definition string
**/
func (s *Model) AddAfterUpdate(name string, definition string) {
	bt := []byte(definition)
	idx := slices.IndexFunc(s.AfterUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterUpdates[idx].Definition = bt
	} else {
		s.AfterUpdates = append(s.AfterUpdates, &Trigger{Name: name, Definition: bt})
	}
}

/**
* AddBeforeDelete
* @param name string, definition string
**/
func (s *Model) AddBeforeDelete(name string, definition string) {
	bt := []byte(definition)
	idx := slices.IndexFunc(s.BeforeDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeDeletes[idx].Definition = bt
	} else {
		s.BeforeDeletes = append(s.BeforeDeletes, &Trigger{Name: name, Definition: bt})
	}
}

/**
* AddAfterDelete
* @param name string, definition string
**/
func (s *Model) AddAfterDelete(name string, definition string) {
	bt := []byte(definition)
	idx := slices.IndexFunc(s.AfterDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterDeletes[idx].Definition = bt
	} else {
		s.AfterDeletes = append(s.AfterDeletes, &Trigger{Name: name, Definition: bt})
	}
}

/**
* Empty: Empties the model
* @return error
**/
func (s *Model) Empty() error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.Required = make([]string, 0)
	s.Hidden = make([]string, 0)
	s.Details = make(map[string]*Detail, 0)
	s.Rollups = make(map[string]*Detail, 0)
	s.Relations = make(map[string]*Detail, 0)
	s.Calcs = make(map[string][]byte, 0)
	s.BeforeInserts = make([]*Trigger, 0)
	s.AfterInserts = make([]*Trigger, 0)
	s.BeforeUpdates = make([]*Trigger, 0)
	s.AfterUpdates = make([]*Trigger, 0)
	s.BeforeDeletes = make([]*Trigger, 0)
	s.AfterDeletes = make([]*Trigger, 0)
	s.btrees = make(map[string]*BTree, 0)

	return nil
}

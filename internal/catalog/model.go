package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/josefina/internal/msg"
	"github.com/cgalvisleon/josefina/internal/store"
)

var (
	ErrorFieldNotFound = errors.New(msg.MSG_FIELD_NOT_FOUND)
)

/**
* From struct
* Define the source of the data, atrib address is the path to the data (host:port)
**/
type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
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

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type Model struct {
	Database      string                      `json:"database"`     // Database name
	Schema        string                      `json:"schema"`       // Schema name
	Name          string                      `json:"name"`         // Model name
	IsInit        bool                        `json:"-"`            // Is initialized
	Path          string                      `json:"path"`         // Path to the model
	Fields        map[string]*Field           `json:"fields"`       // Fields
	Indexes       []string                    `json:"indexes"`      // Indexes
	PrimaryKeys   []string                    `json:"primary_keys"` // Primary keys
	ForeignKeys   map[string]*Detail          `json:"foreign_keys"` // Foreign keys
	Unique        []string                    `json:"unique"`       // Unique
	Required      []string                    `json:"required"`     // Required
	Hidden        []string                    `json:"hidden"`       // Hidden
	Details       map[string]*Detail          `json:"details"`      // Details
	Rollups       map[string]*Detail          `json:"rollups"`      // Rollups
	Relations     map[string]*Detail          `json:"relations"`    // Relations
	Calcs         map[string][]byte           `json:"calcs"`        // Calculated fields
	BeforeInserts []*Trigger                  `json:"-"`            // Before insert triggers
	AfterInserts  []*Trigger                  `json:"-"`            // After insert triggers
	BeforeUpdates []*Trigger                  `json:"-"`            // Before update triggers
	AfterUpdates  []*Trigger                  `json:"-"`            // After update triggers
	BeforeDeletes []*Trigger                  `json:"-"`            // Before delete triggers
	AfterDeletes  []*Trigger                  `json:"-"`            // After delete triggers
	Version       int                         `json:"version"`      // Version
	IsCore        bool                        `json:"is_core"`      // Is core model
	IsStrict      bool                        `json:"is_strict"`    // Is strict model
	stores        map[string]*store.FileStore `json:"-"`            // Stores
	btrees        map[string]*BTree           `json:"-"`            // Secondary indexes (B+ tree, self-persisting)
	schema        *Schema                     `json:"-"`            // Schema
	vm            *js.VM                      `json:"-"`            // Virtual machine
	storeVm       js.Store                    `json:"-"`            // Virtual machine
	mode          store.Mode                  `json:"-"`            // Mode
	mu            sync.RWMutex                `json:"-"`            // Mutex
	isDebug       bool                        `json:"-"`            // Is debug
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
* GenKey: Returns a new key for the model
* @return string
**/
func (s *Model) GenKey() string {
	return reg.GenUUId(s.Name)
}

/**
* From: Returns a new From instance
* @return *From
**/
func (s *Model) From() *From {
	return &From{
		Database: s.Database,
		Schema:   s.Schema,
		Name:     s.Name,
	}
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
	for _, name := range s.Indexes {
		if name == INDEX {
			if _, err := s.Store(INDEX); err != nil {
				return err
			}
			continue
		}
		if _, err := s.indexBTree(name); err != nil {
			return err
		}
	}

	s.IsInit = true
	return nil
}

/**
* indexBTree returns the BTree for field, opening and loading it on first access.
* @param field string
* @return *BTree, error
**/
func (s *Model) indexBTree(field string) (*BTree, error) {
	s.mu.RLock()
	bt, exists := s.btrees[field]
	s.mu.RUnlock()
	if exists {
		return bt, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	bt, err := OpenBTree(s.Path, field)
	if err != nil {
		return nil, err
	}

	s.btrees[field] = bt
	return bt, nil
}

/**
* Store: Opens a store
* @param name string
* @return *store.FileStore, error
**/
func (s *Model) Store(name string) (*store.FileStore, error) {
	s.mu.RLock()
	result, exists := s.stores[name]
	s.mu.RUnlock()
	if exists {
		return result, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Re-check after acquiring write lock to avoid duplicate opens.
	if result, exists = s.stores[name]; exists {
		return result, nil
	}

	result, err := store.Open(s.Path, name, s.mode)
	if err != nil {
		return nil, err
	}

	if s.isDebug {
		result = result.IsDebug()
	}

	s.stores[name] = result

	return result, nil
}

/**
* Source: Returns the primary store
* @return *store.FileStore, error
**/
func (s *Model) Source() (*store.FileStore, error) {
	return s.Store(INDEX)
}

/**
* Put: Puts a raw value by primary key
* @param idx string, value any
* @return error
**/
func (s *Model) Put(idx string, value any) error {
	source, err := s.Source()
	if err != nil {
		return err
	}

	return source.Put(idx, value)
}

/**
* Get: Gets a document by primary key
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

	return exists, nil
}

/**
* Remove: Removes a document by primary key (no index cleanup)
* @param idx string
* @return error
**/
func (s *Model) Remove(idx string) error {
	source, err := s.Source()
	if err != nil {
		return err
	}

	_, err = source.Delete(idx)
	return err
}

/**
* putObject: Inserts or updates a document and keeps secondary indexes in sync.
* @param idx string, object et.Json
* @return error
**/
func (s *Model) putObject(idx string, object et.Json) error {
	object[INDEX] = idx

	// Save document to primary store.
	source, err := s.Source()
	if err != nil {
		return err
	}
	if err := source.Put(idx, object); err != nil {
		return err
	}

	// Update each secondary BTree index.
	for _, name := range s.Indexes {
		if name == INDEX {
			continue
		}
		v := object[name]
		if v == nil {
			continue
		}
		bt, err := s.indexBTree(name)
		if err != nil {
			return err
		}
		if err := bt.Insert(KeyFromAny(v), idx); err != nil {
			return err
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
	// Remove from primary store.
	source, err := s.Source()
	if err != nil {
		return err
	}

	exists := source.IsExist(idx)
	if !exists {
		return nil
	}

	if _, err := source.Delete(idx); err != nil {
		return err
	}

	// Remove from each secondary BTree index.
	for _, name := range s.Indexes {
		if name == INDEX {
			continue
		}
		v := current[name]
		if v == nil {
			continue
		}
		bt, err := s.indexBTree(name)
		if err != nil {
			return err
		}
		if _, err := bt.Delete(KeyFromAny(v), idx); err != nil {
			return err
		}
	}

	return nil
}

/**
* fireTriggers: Fires the triggers
* @param triggers []*Trigger, old, new et.Json
* @return error
**/
func (s *Model) fireTriggers(trigger *Trigger, old, new *et.Json, tx *Tx) (*Tx, error) {
	var err error
	s.vm, err = js.New(fmt.Sprintf("trigger_%s", trigger.Name))
	if err != nil {
		return tx, err
	}

	s.vm.Set("Old", old)
	s.vm.Set("New", new)
	s.vm.Set("Tx", tx)
	_, err = s.vm.Run(string(trigger.Definition))
	if err != nil {
		return tx, err
	}

	return tx, nil
}

/**
* IsExists: Checks if a document exists by primary key
* @param idx string
* @return bool, error
**/
func (s *Model) IsExists(idx string) (bool, error) {
	source, err := s.Source()
	if err != nil {
		return false, err
	}

	return source.IsExist(idx), nil
}

/**
* insert: Inserts a document and keeps secondary indexes in sync.
* @param idx string, new et.Json, tx *Tx
* @return (*Transaction, error)
**/
func (s *Model) insert(idx string, new et.Json, tx *Tx) (*Tx, error) {
	tx = GetTx(s.schema.db, tx)
	var old = et.Json{}
	for _, trigger := range s.BeforeInserts {
		tx, err := s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	tx.Add(s, INSERT, idx, old, new)

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
* @param idx string, new et.Json
* @return (*Transaction, error)
**/
func (s *Model) Insert(idx string, new et.Json, tx *Tx) (*Tx, error) {
	exists, err := s.IsExists(idx)
	if err != nil {
		return tx, err
	}
	if exists {
		return tx, errors.New(msg.MSG_RECORD_EXISTS)
	}

	return s.insert(idx, new, tx)
}

/**
* Update: Updates a document and keeps secondary indexes in sync.
* @param idx string, new et.Json
* @return (et.Json, error)
**/
func (s *Model) Update(idx string, new et.Json, tx *Tx) (*Tx, error) {
	tx = GetTx(s.schema.db, tx)
	old := et.Json{}
	exists, err := s.Get(idx, &old)
	if err != nil {
		return tx, err
	}
	if !exists {
		return tx, fmt.Errorf(msg.MSG_RECORD_NOT_FOUND)
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
* @param idx string, new et.Json
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
	old := et.Json{}
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

	tx.Add(s, DELETE, idx, old, new)

	for _, trigger := range s.AfterDeletes {
		tx, err := s.fireTriggers(trigger, &old, &new, tx)
		if err != nil {
			return tx, err
		}
	}

	return tx, nil
}

/**
* Equal: Gets the model as object
* @param idx string
* @return et.Json, error
**/
func (s *Model) Equal(idx string) (et.Json, error) {
	var dest et.Json
	exists, err := s.Get(idx, &dest)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New(msg.MSG_RECORD_NOT_FOUND)
	}
	return dest, nil
}

/**
* EqualByIndex: Returns all primary keys where field == key.
* @param field string, key IndexKey
* @return []string, bool
**/
func (s *Model) EqualByIndex(field string, key IndexKey) ([]string, bool) {
	bt, err := s.indexBTree(field)
	if err != nil {
		return nil, false
	}
	return bt.Equal(key)
}

/**
* NotEqualByIndex: Returns all primary keys where field != key.
* @param field string, key IndexKey
* @return []string
**/
func (s *Model) NotEqualByIndex(field string, key IndexKey) []string {
	bt, err := s.indexBTree(field)
	if err != nil {
		return nil
	}
	return bt.NotEqual(key)
}

/**
* BetweenByIndex: Returns all primary keys where field is in [from, to] inclusive.
* Pass IndexKey{} for open bounds.
* @param field string, from, to IndexKey, asc bool
* @return []string
**/
func (s *Model) BetweenByIndex(field string, from, to IndexKey, asc bool) []string {
	bt, err := s.indexBTree(field)
	if err != nil {
		return nil
	}
	return bt.Between(from, to, asc)
}

/**
* MoreByIndex: Returns all primary keys where field > key.
* @param field string, key IndexKey, asc bool
* @return []string
**/
func (s *Model) MoreByIndex(field string, key IndexKey, asc bool) []string {
	bt, err := s.indexBTree(field)
	if err != nil {
		return nil
	}
	return bt.More(key, asc)
}

/**
* MoreEqByIndex: Returns all primary keys where field >= key.
* @param field string, key IndexKey, asc bool
* @return []string
**/
func (s *Model) MoreEqByIndex(field string, key IndexKey, asc bool) []string {
	bt, err := s.indexBTree(field)
	if err != nil {
		return nil
	}
	return bt.MoreEq(key, asc)
}

/**
* LessByIndex: Returns all primary keys where field < key.
* @param field string, key IndexKey, asc bool
* @return []string
**/
func (s *Model) LessByIndex(field string, key IndexKey, asc bool) []string {
	bt, err := s.indexBTree(field)
	if err != nil {
		return nil
	}
	return bt.Less(key, asc)
}

/**
* LessEqByIndex: Returns all primary keys where field <= key.
* @param field string, key IndexKey, asc bool
* @return []string
**/
func (s *Model) LessEqByIndex(field string, key IndexKey, asc bool) []string {
	bt, err := s.indexBTree(field)
	if err != nil {
		return nil
	}
	return bt.LessEq(key, asc)
}

/**
* Exists: Checks if a primary key exists
* @param idx string
* @return bool, error
**/
func (s *Model) Exists(idx string) (bool, error) {
	source, err := s.Store(INDEX)
	if err != nil {
		return false, err
	}

	return source.IsExist(idx), nil
}

/**
* Count: Counts documents in the primary store
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
* ForEachTx: Iterates over all transactions
* @param next func(idx string, tx Tx) (bool, error), asc bool, offset, limit, workers int
* @return error
**/
func (s *Model) ForEachTx(next func(idx string, tx Tx) (bool, error), asc bool, offset, limit, workers int) error {
	st, err := s.Source()
	if err != nil {
		return err
	}

	return st.Iterate(func(idx string, src []byte) (bool, error) {
		var tx Tx
		if err := json.Unmarshal(src, &tx); err != nil {
			return false, err
		}
		return next(idx, tx)
	}, asc, offset, limit, workers)
}

/**
* ForEach: Iterates over all documents in the primary store
* @param next func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int
* @return error
**/
func (s *Model) ForEach(next func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int) error {
	st, err := s.Source()
	if err != nil {
		return err
	}

	return st.Iterate(func(idx string, src []byte) (bool, error) {
		item := et.Json{}
		if err := json.Unmarshal(src, &item); err != nil {
			return false, err
		}
		return next(idx, item)
	}, asc, offset, limit, workers)
}

/**
* OnIndex
* @param fn store.SetIndexFn
**/
func (s *Model) OnIndex(name string, fn store.SetIndexFn) {
	source, err := s.Store(name)
	if err != nil {
		return
	}
	source.OnIndex(fn)
}

/**
* OnPut
* @param name string, fn store.Putfn
* @return error
**/
func (s *Model) OnPut(name string, fn store.Putfn) error {
	source, err := s.Store(name)
	if err != nil {
		return err
	}
	source.OnPut(fn)
	return nil
}

/**
* OnDelete
* @param name string, fn store.Deletefn
* @return error
**/
func (s *Model) OnDelete(name string, fn store.Deletefn) error {
	source, err := s.Store(name)
	if err != nil {
		return err
	}
	source.OnDelete(fn)
	return nil
}

/**
* CreateIndex: Creates an index for the specified field
* @param name string
* @return error
**/
func (s *Model) CreateIndex(name string) error {
	exists, err := s.DefineIndex(name)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return s.ForEach(func(idx string, item et.Json) (bool, error) {
		v := item[name]
		if v == nil {
			return true, nil
		}
		bt, err := s.indexBTree(name)
		if err != nil {
			return true, err
		}
		if err := bt.Insert(KeyFromAny(v), idx); err != nil {
			return true, err
		}
		return true, nil
	}, true, 0, 0, 1)
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
	s.Indexes = make([]string, 0)
	s.PrimaryKeys = make([]string, 0)
	s.ForeignKeys = make(map[string]*Detail, 0)
	s.Unique = make([]string, 0)
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

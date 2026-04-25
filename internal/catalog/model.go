package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
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
	btrees        map[string]*BTree           `json:"-"`            // In-memory secondary indexes (B+ tree)
	schema        *Schema                     `json:"-"`            // Schema
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
		_, err := s.Store(name)
		if err != nil {
			return err
		}
	}

	if err := s.rebuildBTrees(); err != nil {
		return err
	}

	s.IsInit = true
	return nil
}

// rebuildBTrees loads each secondary BTree from its persisted FileStore.
// Each secondary FileStore entry is: fieldValue → map[string]bool{pk: true}.
// This is O(distinct values) per index, much faster than scanning all documents.
func (s *Model) rebuildBTrees() error {
	for _, name := range s.Indexes {
		if name == INDEX {
			continue
		}
		st, err := s.Store(name)
		if err != nil {
			return err
		}
		bt := s.indexTree(name)
		if err := st.Iterate(func(fieldVal string, data []byte) (bool, error) {
			var pks map[string]bool
			if err := json.Unmarshal(data, &pks); err != nil {
				return true, nil
			}
			key := s.keyFromField(name, fieldVal)
			for pk := range pks {
				bt.Insert(key, pk)
			}
			return true, nil
		}, true, 0, 0, 1); err != nil {
			return err
		}
	}
	return nil
}

/**
* indexTree returns the BTree for field, creating it on first access.
* @param field string
* @return *BTree
**/
func (s *Model) indexTree(field string) *BTree {
	s.mu.RLock()
	bt, exists := s.btrees[field]
	s.mu.RUnlock()
	if exists {
		return bt
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bt = NewBTree()
	s.btrees[field] = bt
	return bt
}

/**
* keyFromField converts a string value to the right IndexKey type using the field schema.
* @param field string, strVal string
* @return IndexKey
**/
func (s *Model) keyFromField(field, strVal string) IndexKey {
	f, exists := s.Fields[field]
	if !exists {
		return KeyString(strVal)
	}
	switch f.TypeData {
	case TpInt, TpAutoIncrement:
		var n int64
		if _, err := fmt.Sscanf(strVal, "%d", &n); err == nil {
			return KeyInt(n)
		}
	case TpFloat:
		var f float64
		if _, err := fmt.Sscanf(strVal, "%g", &f); err == nil {
			return KeyFloat(f)
		}
	case TpBoolean:
		switch strVal {
		case "true", "1":
			return KeyBool(true)
		case "false", "0":
			return KeyBool(false)
		}
	case TpDateTime:
		if t, err := time.Parse(time.RFC3339, strVal); err == nil {
			return KeyDateTime(t)
		}
	}
	return KeyString(strVal)
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
* Source: Returns the source
* @return *store.FileStore, error
**/
func (s *Model) Source() (*store.FileStore, error) {
	result, err := s.Store(INDEX)
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
* PutObject: Puts the model
* @param idx string, object et.Json
* @return error
**/
func (s *Model) PutObject(idx string, object et.Json) error {
	object[INDEX] = idx
	for _, name := range s.Indexes {
		v := object[name]
		key := fmt.Sprintf("%v", v)
		if key == "" {
			continue
		}

		st, err := s.Store(name)
		if err != nil {
			return err
		}

		// Persist primary key in FileStore
		if name == INDEX {
			if err := st.Put(key, object); err != nil {
				return err
			}
			continue
		}

		// Persist inverted index in FileStore
		index := map[string]bool{}
		if _, err := st.Get(key, &index); err != nil {
			return err
		}
		index[idx] = true
		if err := st.Put(key, index); err != nil {
			return err
		}

		// Update in-memory BTree
		s.indexTree(name).Insert(KeyFromAny(v), idx)
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
		v := data[name]
		key := fmt.Sprintf("%v", v)
		if key == "" {
			continue
		}

		st, err := s.Store(name)
		if err != nil {
			return err
		}

		if name == INDEX {
			if _, err := st.Delete(key); err != nil {
				return err
			}
			continue
		}

		// Update persisted inverted index
		index := map[string]bool{}
		if found, err := st.Get(key, &index); err != nil {
			return err
		} else if found {
			delete(index, idx)
			if len(index) == 0 {
				if _, err := st.Delete(key); err != nil {
					return err
				}
			} else {
				if err := st.Put(key, index); err != nil {
					return err
				}
			}
		}

		// Update in-memory BTree
		s.indexTree(name).Delete(KeyFromAny(v), idx)
	}
	return nil
}

/**
* GetIndex: Gets the index
* @param field, key string, dest map[string]bool
* @return bool, error
**/
func (s *Model) GetIndex(field, key string, dest map[string]bool) (bool, error) {
	pks, exists := s.indexTree(field).Get(s.keyFromField(field, key))
	if !exists {
		return false, nil
	}
	for _, pk := range pks {
		dest[pk] = true
	}
	return true, nil
}

/**
* BetweenIndex returns all primary keys where field is in [from, to] (inclusive).
* Pass IndexKey{} for open bounds.
* @param field string, from, to IndexKey, asc bool
* @return []string
**/
func (s *Model) BetweenIndex(field string, from, to IndexKey, asc bool) []string {
	return s.indexTree(field).Between(from, to, asc)
}

/**
* EqualIndex returns all primary keys where field equals key.
* @param field string, key IndexKey
* @return []string, bool
**/
func (s *Model) EqualIndex(field string, key IndexKey) ([]string, bool) {
	return s.indexTree(field).Get(key)
}

func (s *Model) MoreIndex(field string, key IndexKey, asc bool) []string {
	return s.indexTree(field).More(key, asc)
}

/**
* MoreEq returns all primary keys where field >= key.
* @param field string, key IndexKey, asc bool
* @return []string
**/
func (s *Model) MoreEqIndex(field string, key IndexKey, asc bool) []string {
	return s.indexTree(field).MoreEq(key, asc)
}

/**
* LessIndex returns all primary keys where field < key.
* @param field string, key IndexKey, asc bool
* @return []string
**/
func (s *Model) LessIndex(field string, key IndexKey, asc bool) []string {
	return s.indexTree(field).Less(key, asc)
}

/**
* LessEqIndex returns all primary keys where field <= key.
* @param field string, key IndexKey, asc bool
* @return []string
**/
func (s *Model) LessEqIndex(field string, key IndexKey, asc bool) []string {
	return s.indexTree(field).LessEq(key, asc)
}

/**
* NotEqualIndex returns all primary keys where field != key.
* @param field string, key IndexKey
* @return []string
**/
func (s *Model) NotEqualIndex(field string, key IndexKey) []string {
	return s.indexTree(field).NotEqual(key)
}

/**
* IsExisted: Check if index exists in model
* @param name string, key string
* @return bool, error
**/
func (s *Model) IsExisted(field, idx string) (bool, error) {
	source, err := s.Store(field)
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
* Iterate: Iterates over the model
* @param fn func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int
* @return bool, error
**/
func (s *Model) Iterate(next func(idx string, item et.Json) (bool, error), asc bool, offset, limit, workers int) error {
	st, err := s.Source()
	if err != nil {
		return err
	}

	err = st.Iterate(func(idx string, src []byte) (bool, error) {
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
* @param name string, definition string
* @return void
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
* @return void
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
* @return void
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
* @return void
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
* @return void
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
* @return void
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
		err := store.Empty()
		if err != nil {
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

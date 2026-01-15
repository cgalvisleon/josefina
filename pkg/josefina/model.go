package josefina

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/cgalvisleon/josefina/pkg/store"
)

var (
	errorRecordNotFound = errors.New(msg.MSG_RECORD_NOT_FOUND)
)

type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
}

/**
* getDb: Returns the database
* @return *DB
**/
func (s *From) getDb() (*DB, error) {
	result, err := GetDB(s.Database)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* getSchema: Returns the schema
* @return *Schema, error
**/
func (s *From) getSchema() (*Schema, error) {
	db, err := s.getDb()
	if err != nil {
		return nil, err
	}

	return db.getSchema(s.Schema)
}

/**
* getModel: Returns the model
* @return *Model, error
**/
func (s *From) getModel() (*Model, error) {
	schema, err := s.getSchema()
	if err != nil {
		return nil, err
	}

	return schema.getModel(s.Name)
}

/**
* getPath: Returns the path
* @return string, error
**/
func (s *From) getPath() (string, error) {
	db, err := s.getDb()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", db.Path, s.Schema), nil
}

type Model struct {
	*From         `json:"from"`
	Fields        map[string]*Field           `json:"fields"`
	Indexes       []string                    `json:"indexes"`
	PrimaryKeys   []string                    `json:"primary_keys"`
	Unique        []string                    `json:"unique"`
	Required      []string                    `json:"required"`
	Hidden        []string                    `json:"hidden"`
	References    []string                    `json:"references"`
	Master        map[string]*Master          `json:"master"`
	Details       map[string]*Detail          `json:"details"`
	Rollups       map[string]*Detail          `json:"rollups"`
	Relations     map[string]*Detail          `json:"relations"`
	BeforeInserts []*Trigger                  `json:"before_inserts"`
	BeforeUpdates []*Trigger                  `json:"before_updates"`
	BeforeDeletes []*Trigger                  `json:"before_deletes"`
	AfterInserts  []*Trigger                  `json:"after_inserts"`
	AfterUpdates  []*Trigger                  `json:"after_updates"`
	AfterDeletes  []*Trigger                  `json:"after_deletes"`
	IsStrict      bool                        `json:"is_strict"`
	Version       int                         `json:"version"`
	IsCore        bool                        `json:"is_core"`
	IsDebug       bool                        `json:"-"`
	db            *DB                         `json:"-"`
	isInit        bool                        `json:"-"`
	data          map[string]*store.FileStore `json:"-"`
	triggers      map[string]*Vm              `json:"-"`
}

/**
* prepared: Prepares the model
* @return error
**/
func (s *Model) prepared() error {
	if len(s.Fields) == 0 {
		s.defineIndexField()
	}

	return nil
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
		fStore, err := store.Open(path, name, s.IsDebug)
		if err != nil {
			return err
		}
		s.data[name] = fStore
	}

	s.isInit = true
	return nil
}

/**
* save: Saves the model
* @param data et.Json
* @return error
**/
func (s *Model) save(data et.Json) error {
	return nil
}

/**
* count: Counts the model
* @return int
**/
func (s *Model) count() int {
	data, ok := s.data[INDEX]
	if !ok {
		return 0
	}

	return data.Count()
}

/**
* getJid: Gets the jid
* @return string
**/
func (s *Model) getJid() string {
	return reg.GenULID(s.Name)
}

/**
* insert: Inserts the model
* @param ctx *Tx, new et.Json
* @return et.Items, error
**/
func (s *Model) insert(ctx *Tx, new et.Json) (et.Items, error) {
	idx, ok := new[INDEX]
	if !ok {
		idx = s.getJid()
		new[INDEX] = idx
	}

	// Validate required fields
	for _, name := range s.Required {
		if _, ok := new[name]; !ok {
			return et.Items{}, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
	}

	// Validate unique fields
	for _, name := range s.Unique {
		if _, ok := new[name]; !ok {
			return et.Items{}, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
		source := s.data[name]
		key := fmt.Sprintf("%v", new[name])
		if source.IsExist(key) {
			return et.Items{}, fmt.Errorf(msg.MSG_RECORD_EXISTS)
		}
	}

	// Run before insert triggers
	for _, trigger := range s.BeforeInserts {
		err := s.runTrigger(trigger, ctx, et.Json{}, new)
		if err != nil {
			return et.Items{}, err
		}
	}

	// Insert data into indexes
	for _, name := range s.Indexes {
		source := s.data[name]
		key := fmt.Sprintf("%v", new[name])
		if key == "" {
			continue
		}
		if name == INDEX {
			source.Put(key, new)
		} else {
			source.Put(key, idx)
		}
	}

	// Run after insert triggers
	for _, trigger := range s.AfterInserts {
		err := s.runTrigger(trigger, ctx, et.Json{}, new)
		if err != nil {
			return et.Items{}, err
		}
	}

	result := et.Items{}
	result.Add(new)
	return result, nil
}

/**
* update: Updates the model
* @param ctx *Tx, data et.Json, where et.Json
* @return et.Items, error
**/
func (s *Model) update(ctx *Tx, data et.Json, where et.Json) (et.Items, error) {
	result := et.Items{}
	selects, err := s.selects(ctx, where)
	if err != nil {
		return result, err
	}

	for _, old := range selects.Result {
		new := old.Clone()
		for k, v := range data {
			new[k] = v
		}

		idx, ok := new[INDEX]
		if !ok {
			return result, errorRecordNotFound
		}

		// Run before update triggers
		for _, trigger := range s.BeforeUpdates {
			err := s.runTrigger(trigger, ctx, old, new)
			if err != nil {
				return et.Items{}, err
			}
		}

		// Insert data into indexes
		for _, name := range s.Indexes {
			source := s.data[name]
			key := fmt.Sprintf("%v", new[name])
			if key == "" {
				continue
			}
			if name == INDEX {
				source.Put(key, new)
			} else {
				source.Put(key, idx)
			}
		}

		// Run after insert triggers
		for _, trigger := range s.AfterInserts {
			err := s.runTrigger(trigger, ctx, old, new)
			if err != nil {
				return et.Items{}, err
			}
		}

		result.Add(new)
	}
	return result, nil
}

/**
* delete: Deletes the model
* @param ctx *Tx, where et.Json
* @return et.Items, error
**/
func (s *Model) delete(ctx *Tx, where et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* upsert: Upserts the model
* @param ctx *Tx, data et.Json
* @return et.Item, error
**/
func (s *Model) upsert(ctx *Tx, data et.Json) (et.Item, error) {
	return et.Item{}, nil
}

/**
* selects: Selects the model
* @param ctx *Tx, query et.Json
* @return et.Items, error
**/
func (s *Model) selects(ctx *Tx, query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

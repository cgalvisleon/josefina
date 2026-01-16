package rds

import (
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
)

type From struct {
	Database string            `json:"database"`
	Schema   string            `json:"schema"`
	Name     string            `json:"name"`
	Fields   map[string]*Field `json:"fields"`
	IsStrict bool              `json:"is_strict"`
	as       string            `json:"-"`
}

/**
* clone: Clones the from
* @return *From
**/
func (s *From) clone() *From {
	return &From{
		Database: s.Database,
		Schema:   s.Schema,
		Name:     s.Name,
		Fields:   s.Fields,
		IsStrict: s.IsStrict,
		as:       s.Name,
	}
}

/**
* setAs
* @param as string
* @return void
**/
func (s *From) setAs(as string) {
	s.as = as
}

/**
* As
* @return string
**/
func (s *From) As() string {
	if s.Schema == "" {
		return s.Name
	}
	return fmt.Sprintf("%s.%s", s.Schema, s.Name)
}

/**
* getField
* @param name string
* @return *Field
**/
func (s *From) getField(name string) *Field {
	result, ok := s.Fields[name]
	if ok {
		return result
	}

	if s.IsStrict {
		return nil
	}

	result, err := newField(s, name, TpAtrib, TpAny, "")
	if err != nil {
		return nil
	}

	return result
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
	References    []string                    `json:"references"`
	Master        map[string]*Detail          `json:"master"`
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
		s.definePrimaryKeys(INDEX)
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
* Stricted: Sets the model to strict
* @return void
**/
func (s *Model) Stricted() {
	s.IsStrict = true
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
* insert: Inserts the model
* @param ctx *Tx, data et.Json
* @return et.Items, error
**/
func (s *Model) insert(ctx *Tx, data et.Json) (et.Items, error) {
	return newCmd(s, insert).insert(ctx, data)
}

/**
* update: Updates the model
* @param ctx *Tx, data et.Json, where *Wheres
* @return et.Items, error
**/
func (s *Model) update(ctx *Tx, data et.Json, where *Wheres) (et.Items, error) {
	return newCmd(s, update).update(ctx, data, where)
}

/**
* delete: Deletes the model
* @param ctx *Tx, where *Wheres
* @return et.Items, error
**/
func (s *Model) delete(ctx *Tx, where *Wheres) (et.Items, error) {
	return newCmd(s, delete).delete(ctx, where)
}

/**
* upsert: Upserts the model
* @param ctx *Tx, data et.Json
* @return et.Items, error
**/
func (s *Model) upsert(ctx *Tx, data et.Json) (et.Items, error) {
	return newCmd(s, upsert).upsert(ctx, data)
}

func (s *Model) byWhere(ctx *Tx, selects, hiddens, ordersAsc, ordersDesc []string, where *Wheres) (et.Items, error) {
	return newCmd(s, select).byWhere(ctx, where)
}
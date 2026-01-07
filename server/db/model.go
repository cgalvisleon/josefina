package db

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/josefina/server/msg"
	"github.com/cgalvisleon/josefina/server/store"
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
	result, ok := dbs[s.Database]
	if !ok {
		return nil, errors.New(msg.MSG_DB_NOT_FOUND)
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

	return fmt.Sprintf("%s/%s/%s", db.Path, s.Database, s.Schema), nil
}

type Model struct {
	*From         `json:"from"`
	Data          *store.FileStore            `json:"data"`
	Fields        []*Field                    `json:"fields"`
	Indexes       map[string]*store.FileStore `json:"indexes"`
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
	Host          string                      `json:"-"`
	IsCore        bool                        `json:"is_core"`
	IsDebug       bool                        `json:"-"`
	isInit        bool                        `json:"-"`
	vm            *Vm                         `json:"-"`
}

/**
* prepared: Prepares the model
* @return error
**/
func (s *Model) prepared() error {
	if len(s.Fields) == 0 {
		s.defineKeyField()
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
	s.Data, err = store.Open(path, s.Name, s.IsDebug)
	if err != nil {
		return err
	}

	s.isInit = true
	return nil
}

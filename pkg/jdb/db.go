package jdb

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var dbs *Model

/**
* initDbs: Initializes the dbs model
* @return error
**/
func initDbs() error {
	if dbs != nil {
		return nil
	}

	db, ok := node.dbs[packageName]
	if !ok {
		path := envar.GetStr("DATA_PATH", "./data")
		db = &DB{
			Name:    packageName,
			Version: Version,
			Path:    fmt.Sprintf("%s/%s", path, packageName),
			Schemas: make(map[string]*Schema, 0),
		}
		node.dbs[packageName] = db
	}

	var err error
	dbs, err = db.newModel("", "dbs", true, 1)
	if err != nil {
		return err
	}
	if err := dbs.init(); err != nil {
		return err
	}

	return nil
}

type DB struct {
	Name     string             `json:"name"`
	Version  string             `json:"version"`
	Path     string             `json:"path"`
	Schemas  map[string]*Schema `json:"schemas"`
	IsStrict bool               `json:"is_strict"`
}

/**
* serialize
* @return []byte, error
**/
func (s *DB) serialize() ([]byte, error) {
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
func (s *DB) ToJson() (et.Json, error) {
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
* save
* @return error
**/
func (s *DB) save() error {
	return node.saveDb(s)
}

/**
* getSchema: Returns a schema by name
* @param name string
* @return *Schema
**/
func (s *DB) getSchema(name string) *Schema {
	name = utility.Normalize(name)
	result, ok := s.Schemas[name]
	if ok {
		return result
	}

	result = &Schema{
		Database: s.Name,
		Name:     name,
		Models:   make(map[string]*Model, 0),
		db:       s,
	}
	s.Schemas[name] = result

	return result
}

/**
* newModel: Creates a new model
* @param schema, name	string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) newModel(schema, name string, isCore bool, version int) (*Model, error) {
	sch := s.getSchema(schema)
	model, err := sch.newModel(name, isCore, version)
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* getDb: Returns a database by name
* @param name string
* @return *DB, error
**/
func getDb(name string) (*DB, error) {
	if !node.started {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	leader := node.getLeader()
	if leader != node.host && leader != "" {
		result, err := methods.getDb(leader, name)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	type Result struct {
		result *DB
		err    error
	}

	ch := make(chan Result)
	go func() {
		name = utility.Normalize(name)
		result, ok := node.dbs[name]
		if ok {
			ch <- Result{result: result, err: nil}
			return
		}

		err := initDbs()
		if err != nil {
			ch <- Result{result: nil, err: err}
			return
		}

		exists, err := dbs.get(name, &result)
		if err != nil {
			ch <- Result{result: nil, err: err}
			return
		}

		if exists {
			ch <- Result{result: result, err: nil}
			return
		}

		path := envar.GetStr("DATA_PATH", "./data")
		result = &DB{
			Name:    name,
			Version: Version,
			Path:    fmt.Sprintf("%s/%s", path, name),
			Schemas: make(map[string]*Schema, 0),
		}
		node.dbs[name] = result

		ch <- Result{result: result, err: nil}
	}()

	res := <-ch
	return res.result, res.err
}

/**
* createDb: Creates a new database
* @param name string, isStrict bool
* @return *DB, error
**/
func CreateDb(name string, isStrict bool) (*DB, error) {
	db, err := getDb(name)
	if err != nil {
		return nil, err
	}

	db.IsStrict = isStrict
	db.save()

	return db, nil
}

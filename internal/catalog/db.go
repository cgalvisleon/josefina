package catalog

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type DB struct {
	Name     string             `json:"name"`
	Version  string             `json:"version"`
	Path     string             `json:"path"`
	Schemas  map[string]*Schema `json:"schemas"`
	IsStrict bool               `json:"is_strict"`
	isDebug  bool               `json:"-"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *DB) Serialize() ([]byte, error) {
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
* SetDebug
* @param debug bool
**/
func (s *DB) SetDebug(debug bool) {
	s.isDebug = debug
	for _, schema := range s.Schemas {
		for _, model := range schema.Models {
			model.isDebug = debug
		}
	}
}

/**
* SetStrict
* @param strict bool
**/
func (s *DB) SetStrict(strict bool) {
	s.IsStrict = strict
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
		Models:   make(map[string]*From, 0),
		db:       s,
	}
	s.Schemas[name] = result

	return result
}

/**
* NewModel: Creates a new model
* @param schema, name	string, isCore bool, version int
* @return *Model, error
**/
func (s *DB) NewModel(schema, name string, isCore bool, version int) (*Model, error) {
	sch := s.getSchema(schema)
	model, err := sch.newModel(name, isCore, version)
	if err != nil {
		return nil, err
	}

	return model, nil
}

/**
* CreateDb: Creates a new database
* @param name string
* @return *DB, error
**/
func CreateDb(name string) (*DB, error) {
	if node == nil {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	name = utility.Normalize(name)
	path := envar.GetStr("DATA_PATH", "./data")
	result := &DB{
		Name:    name,
		Version: node.version,
		Path:    fmt.Sprintf("%s/%s", path, name),
		Schemas: make(map[string]*Schema, 0),
	}
	AddDb(result)

	return result, nil
}

/**
* GetDb: Returns a database by name
* @param name string
* @return bool, error
**/
func GetDb(name string, dest *DB) (bool, error) {
	if node == nil {
		return false, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return false, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	var ok bool
	dest, ok = node.dbs[name]
	if ok {
		return true, nil
	}

	return false, nil
}

/**
* AddDb: Adds a database to the global map
* @param db *DB
**/
func AddDb(db *DB) {
	dbs[db.Name] = db
}

/**
* RemoveDb: Removes a database from the global map
* @param name string
**/
func RemoveDb(name string) {
	delete(dbs, name)
}

/**
* CoreDb: Returns the core database
* @return *DB, error
**/
func CoreDb() (*DB, error) {
	name := "josefina"
	result, ok := dbs[name]
	if ok {
		return result, nil
	}

	return createDb(name)
}

package jdb

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type DB struct {
	Name     string             `json:"name"`
	Path     string             `json:"path"`
	Schemas  map[string]*Schema `json:"schemas"`
	IsStrict bool               `json:"is_strict"`
	isDebug  bool               `json:"-"`
}

/**
* NewDb: Creates a new database
* @param name string
* @return *DB, error
**/
func NewDb(name string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	path := envar.GetStr("DATA_PATH", "./data")
	result := &DB{
		Name:    name,
		Path:    fmt.Sprintf("%s/%s", path, name),
		Schemas: make(map[string]*Schema, 0),
	}

	return result, nil
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
* Debug
**/
func (s *DB) Debug() {
	s.isDebug = true
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

var dbs *Model

/**
* initDbs: Initializes the dbs model
* @return error
**/
func (s *Node) initDbs() error {
	if dbs != nil {
		return nil
	}

	db, err := s.coreDb()
	if err != nil {
		return err
	}

	dbs, err = db.NewModel("", "dbs", true, 1)
	if err != nil {
		return err
	}
	if err := dbs.Init(); err != nil {
		return err
	}

	return nil
}

/**
* coreDb: Creates the core database
* @return *catalog.DB, error
**/
func (s *Node) coreDb() (*catalog.DB, error) {
	name := "josefina"
	s.muDB.RLock()
	result, ok := s.dbs[name]
	s.muDB.RUnlock()
	if ok {
		return result, nil
	}

	result, err := catalog.NewDb(name)
	if err != nil {
		return nil, err
	}
	s.muDB.Lock()
	s.dbs[name] = result
	s.muDB.Unlock()

	return result, nil
}

/**
* GetDb: Gets a model
* @param name string, dest *jdb.Model
* @return bool, error
**/
func (s *Node) GetDb(name string) (*catalog.DB, bool) {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.getDb(name)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.GetDb", name)
		if res.Error != nil {
			return nil, false
		}

		var result *catalog.DB
		var exists bool
		err := res.Get(&result, &exists)
		if err != nil {
			return nil, false
		}

		return result, exists
	}

	return nil, false
}

/**
* CreateDb: Creates a new database
* @param name string
* @return *DB, error
**/
func (s *Node) CreateDb(name string) (*catalog.DB, error) {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.CreateDb(name)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.CreateDb", name)
		if res.Error != nil {
			return nil, res.Error
		}

		var result *catalog.DB
		err := res.Get(&result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	return nil, errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* DropDb: Removes a db
* @param name string
* @return error
**/
func (s *Node) DropDb(name string) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.DropDb(name)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.DropDb", name)
		if res.Error != nil {
			return res.Error
		}

		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

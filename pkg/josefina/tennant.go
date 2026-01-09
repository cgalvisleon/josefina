package josefina

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

const (
	packageName = "josefina"
)

type Tennant struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Path    string         `json:"path"`
	Dbs     map[string]*DB `json:"dbs"`
}

/**
* loadTennant
* @param name string
* @return *Tennant, error
**/
func loadTennant(path, name, version string) (*Tennant, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	result := &Tennant{
		Name:    name,
		Version: version,
		Path:    path,
		Dbs:     make(map[string]*DB),
	}
	err := result.loadCore()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Serialize
* @return []byte, error
**/
func (s *Tennant) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* getDb
* @param name string
* @return *DB, error
**/
func (s *Tennant) getDb(name string) (*DB, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	name = utility.Normalize(name)
	result, ok := s.Dbs[name]
	if ok {
		return result, nil
	}

	result = &DB{
		Name:    name,
		Version: s.Version,
		Path:    fmt.Sprintf("%s/%s", s.Path, name),
		Schemas: make(map[string]*Schema),
		Models:  make(map[string]*Model),
	}
	s.Dbs[name] = result

	return result, nil
}

/**
* getModel
* @param database string, schema string, model string
* @return *Model, error
**/
func (s *Tennant) getModel(database, schema, name string) (*Model, error) {
	db, err := s.getDb(database)
	if err != nil {
		return nil, err
	}

	return db.getModel(schema, name)
}

package rds

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
	*Node
	Nodes []string `json:"nodes"`
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
		Node: &Node{
			Name:    name,
			Version: version,
			Path:    path,
			Dbs:     make(map[string]*DB),
		},
		Nodes: []string{},
	}
	err := result.load()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* load
* @return error
**/
func (s *Tennant) loadDbs() error {
	if databases == nil {
		return fmt.Errorf(msg.MSG_DONT_HAVE_DATABASES)
	}

	st, err := databases.source()
	if err != nil {
		return err
	}

	err = st.Iterate(func(id string, src []byte) (bool, error) {
		var item *DB
		err := json.Unmarshal(src, &item)
		if err != nil {
			return false, err
		}

		err = s.loadDb(item)
		if err != nil {
			return false, err
		}

		return true, nil
	}, true, 0, 0, 1)
	return err
}

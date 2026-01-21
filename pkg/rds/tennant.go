package rds

import (
	"encoding/json"
	"fmt"

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
func loadTennant(path, version string) (*Tennant, error) {
	node, err := newNode(version, path)
	if err != nil {
		return nil, err
	}

	result := &Tennant{
		Node:  node,
		Nodes: []string{},
	}
	err = result.load()
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

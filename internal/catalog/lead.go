package catalog

import (
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Lead struct{}

/**
* CreateDb: Creates a new database
* @param name string
* @return *DB, error
**/
func (s *Lead) CreateDb(name string) (*DB, error) {
	if node == nil {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_INITIALIZED)
	}

	leader, imLeader := node.GetLeader()
	if !imLeader && leader != nil {
		res := node.Request(leader, "Leader.CreateDb", name)
		if res.Error != nil {
			return nil, res.Error
		}

		var result *DB
		err := res.Get(0, &result)
		if err != nil {
			return nil, err
		}

		return result, nil
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

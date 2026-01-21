package rds

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/josefina/pkg/msg"
)

const (
	packageName = "josefina"
)

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

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
* loadTennant
* @param name string
* @return *Tennant, error
**/
func loadMaster(path, version string) (*Node, error) {
	result, err := newNode(Master, version, path)
	if err != nil {
		return nil, err
	}

	db := newDb(result.Path, packageName, result.Version)
	if err := initTransactions(db); err != nil {
		return nil, err
	}
	if err := initDatabases(db); err != nil {
		return nil, err
	}
	if err := initUsers(db); err != nil {
		return nil, err
	}
	if err := initSeries(db); err != nil {
		return nil, err
	}
	if err := initRecords(db); err != nil {
		return nil, err
	}
	if err := initModels(db); err != nil {
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

package jdb

import "github.com/cgalvisleon/et/et"

func loadErrors(db *DB) error {
	result, err := db.NewModel("", "errors", true, 1)
	if err != nil {
		return err
	}

	err = result.Init()
	if err != nil {
		return err
	}

	db.errors = result

	return nil
}

/**
* putError
* @param tag string, err error
* @return error
**/
func (db *DB) putError(tag string, err error) (string, error) {
	idx := db.errors.GenKey()
	_, er := db.errors.Insert(idx, et.Json{
		INDEX:   idx,
		"tag":   tag,
		"error": err.Error(),
	}, nil, 0)
	if er != nil {
		return "", er
	}

	return idx, nil
}

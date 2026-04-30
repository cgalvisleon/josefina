package jdb

import (
	"github.com/cgalvisleon/et/et"
)

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
* @param model, tag, id string, err error
* @return error
**/
func (db *DB) putError(model, tag, id string, err error) (string, error) {
	idx := db.errors.GenKey()
	_, er := db.errors.Insert(idx, et.Json{
		INDEX:   idx,
		"model": model,
		"tag":   tag,
		"id":    id,
		"error": err.Error(),
	}, nil)
	if er != nil {
		return "", er
	}

	return idx, nil
}

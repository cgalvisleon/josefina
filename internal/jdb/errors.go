package jdb

import (
	"github.com/cgalvisleon/et/et"
)

/**
* loadErrors: Initializes the internal errors model for a database.
* @param db *DB
* @return error
**/
func (s *DB) loadErrors() error {
	result, err := s.Define(DModel{
		Schema:  sysSchema,
		Name:    "errors",
		IsCore:  true,
		Version: 1,
	})
	if err != nil {
		return err
	}

	err = result.Init()
	if err != nil {
		return err
	}

	s.errors = result

	return nil
}

/**
* putError
* @param model, tag, id string, err error
* @return error
**/
func (db *DB) putError(model, tag, id string, err error) (string, error) {
	idx := db.errors.GenKey()
	_, er := db.errors.
		Insert(et.Json{
			INDEX:   idx,
			"model": model,
			"tag":   tag,
			"id":    id,
			"error": err.Error(),
		}).
		Exec()
	if er != nil {
		return "", er
	}

	return idx, nil
}

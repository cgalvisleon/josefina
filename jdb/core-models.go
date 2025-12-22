package jdb

import (
	"github.com/cgalvisleon/et/et"
)

var models *Model

/**
* defineModel
* @param db *DB
* @return error
**/
func defineModel(db *DB) error {
	if models != nil {
		return nil
	}

	var err error
	models, err = db.Define(et.Json{
		"schema":  "core",
		"name":    "models",
		"version": 1,
		"columns": []et.Json{
			{
				"name": "created_at",
				"type": "datetime",
			},
			{
				"name": "updated_at",
				"type": "datetime",
			},
			{
				"name": "name",
				"type": "text",
			},
			{
				"name": "version",
				"type": "int",
			},
			{
				"name": "definition",
				"type": "bytes",
			},
			{
				"name": INDEX,
				"type": "key",
			},
		},
		"record_field": INDEX,
		"primary_keys": []string{"name"},
		"indexes":      []string{"version", INDEX},
		"is_core":      true,
	})
	if err != nil {
		return err
	}

	if err = models.Init(); err != nil {
		return err
	}

	return nil
}

/**
* deleteModel
* @param name string
* @return error
**/
func (s *DB) deleteModel(name string) error {
	if models == nil {
		return nil
	}

	_, err := models.
		Delete().
		Where(Eq("name", name)).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

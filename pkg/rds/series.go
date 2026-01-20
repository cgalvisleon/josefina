package rds

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

var series *Model

/**
* initSeries: Initializes the series model
* @param db *DB
* @return error
**/
func initSeries(db *DB) error {
	var err error
	series, err = db.newModel("", "series", true, 1)
	if err != nil {
		return err
	}
	series.DefineAtrib("name", TpText, "")
	series.DefineAtrib("tag", TpText, "")
	series.DefineAtrib("value", TpInt, 0)
	series.DefineAtrib("format", TpText, "")
	series.DefinePrimaryKeys("name", "tag")
	if err := series.init(); err != nil {
		return err
	}

	return nil
}

/**
* CreateSerie: Creates a new serie
* @param name, tag, format string, value int
* @return error
**/
func CreateSerie(name, tag, format string, value int) error {
	if series == nil {
		return errors.New(msg.MSG_SERIES_NOT_FOUND)
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}
	if format == "" {
		format = `%d`
	}

	_, err := series.insert(nil, et.Json{
		"name":   name,
		"tag":    tag,
		"value":  value,
		"format": format,
	})
	return err
}

/**
* UpdateSerie: Updates a serie
* @param name, tag string, value int
* @return error
**/
func UpdateSerie(name, tag string, value int) error {
	if series == nil {
		return errors.New(msg.MSG_SERIES_NOT_FOUND)
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	_, err := series.update(nil,
		et.Json{
			"value": value,
		},
		Where(Eq("name", name)).
			And(Eq("tag", tag)))
	return err
}

func GetSerie(name, tag string) (et.Json, error) {
	if series == nil {
		return et.Json{}, errors.New(msg.MSG_SERIES_NOT_FOUND)
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	result, err := series.get(nil, Where(Eq("name", name)).And(Eq("tag", tag)))
	if err != nil {
		return et.Json{}, err
	}

	if len(result) == 0 {
		return et.Json{}, errors.New(msg.MSG_NOT_FOUND)
	}

	return result[0], nil
}

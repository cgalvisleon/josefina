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

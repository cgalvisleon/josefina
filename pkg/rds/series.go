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
func initSeries() error {
	if series != nil {
		return nil
	}

	db, err := getDb(packageName)
	if err != nil {
		return err
	}

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
* createSerie: Creates a new serie
* @param name, tag, format string, value int
* @return error
**/
func createSerie(name, tag, format string, value int) error {
	if !node.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader, err := node.leader()
	if err != nil {
		return err
	}

	if leader != node.host {
		err := methods.createSerie(leader, name, tag, format, value)
		if err != nil {
			return err
		}

		return nil
	}

	err = initSeries()
	if err != nil {
		return err
	}

	if format == "" {
		format = `%d`
	}

	_, err = series.
		Insert(et.Json{
			"name":   name,
			"tag":    tag,
			"value":  value,
			"format": format,
		}).
		Execute(nil)
	return err
}

/**
* dropSerie: Drops a serie
* @param name, tag string
* @return error
**/
func dropSerie(name, tag string) error {
	if series == nil {
		return errors.New(msg.MSG_SERIES_NOT_FOUND)
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	_, err := series.
		Delete().
		Where(Eq("name", name)).
		And(Eq("tag", tag)).
		Execute(nil)
	return err
}

/**
* setSerie: Sets a serie
* @param name, tag string, value int
* @return error
**/
func setSerie(name, tag string, value int) error {
	if series == nil {
		return errors.New(msg.MSG_SERIES_NOT_FOUND)
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	_, err := series.
		Update(et.Json{
			"value": value,
		}).
		Where(Eq("name", name)).
		And(Eq("tag", tag)).
		Execute(nil)
	return err
}

/**
* getSerie: Gets a serie
* @param name, tag string
* @return et.Json, error
**/
func getSerie(name, tag string) (et.Json, error) {
	if series == nil {
		return et.Json{}, errors.New(msg.MSG_SERIES_NOT_FOUND)
	}
	if !utility.ValidStr(name, 0, []string{""}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return et.Json{}, fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	items, err := series.
		Update(et.Json{}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			value := old.Int("value")
			new["value"] = value + 1
			return nil
		}).
		Where(Eq("name", name)).
		And(Eq("tag", tag)).
		Execute(nil)
	if err != nil {
		return et.Json{}, err
	}

	if len(items) != 1 {
		return et.Json{}, errors.New(msg.MSG_INVALID_CONDITION_ONLY_ONE)
	}

	item := items[0]
	format := item.String("format")
	value := item.Int("value")
	code := fmt.Sprintf(format, value)

	return et.Json{
		"value": value,
		"code":  code,
	}, nil
}

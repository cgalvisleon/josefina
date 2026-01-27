package jdb

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
* @param tag, format string, value int
* @return error
**/
func createSerie(tag, format string, value int) error {
	if !node.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader := node.getLeader()
	if leader != node.host && leader != "" {
		err := methods.createSerie(leader, tag, format, value)
		if err != nil {
			return err
		}

		return nil
	}

	err := initSeries()
	if err != nil {
		return err
	}

	if format == "" {
		format = `%d`
	}

	_, err = series.
		Insert(et.Json{
			"tag":    tag,
			"value":  value,
			"format": format,
		}).
		Execute(nil)
	return err
}

/**
* dropSerie: Drops a serie
* @param tag string
* @return error
**/
func dropSerie(tag string) error {
	if !node.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader := node.getLeader()
	if leader != node.host && leader != "" {
		err := methods.dropSerie(leader, tag)
		if err != nil {
			return err
		}

		return nil
	}

	err := initSeries()
	if err != nil {
		return err
	}

	_, err = series.
		Delete().
		Where(Eq("tag", tag)).
		Execute(nil)
	return err
}

/**
* setSerie: Sets a serie
* @param tag string, value int
* @return error
**/
func setSerie(tag string, value int) error {
	if !node.started {
		return fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader := node.getLeader()
	if leader != node.host && leader != "" {
		err := methods.setSerie(leader, tag, value)
		if err != nil {
			return err
		}

		return nil
	}

	err := initSeries()
	if err != nil {
		return err
	}

	_, err = series.
		Update(et.Json{
			"value": value,
		}).
		Where(Eq("tag", tag)).
		Execute(nil)
	return err
}

/**
* getSerie: Gets a serie
* @param tag string
* @return et.Json, error
**/
func getSerie(tag string) (et.Json, error) {
	if !node.started {
		return nil, fmt.Errorf(msg.MSG_NODE_NOT_STARTED)
	}
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader := node.getLeader()
	if leader != node.host && leader != "" {
		result, err := methods.getSerie(leader, tag)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	err := initSeries()
	if err != nil {
		return nil, err
	}

	items, err := series.
		Update(et.Json{}).
		BeforeUpdateFn(func(tx *Tx, old, new et.Json) error {
			value := old.Int("value")
			new["value"] = value + 1
			return nil
		}).
		Where(Eq("tag", tag)).
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

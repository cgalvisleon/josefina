package core

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var series *catalog.Model

/**
* initSeries: Initializes the series model
* @param db *DB
* @return error
**/
func initSeries() error {
	if series != nil {
		return nil
	}

	db, err := catalog.CoreDb()
	if err != nil {
		return err
	}

	series, err = db.NewModel("", "series", true, 1)
	if err != nil {
		return err
	}
	series.DefineAtrib("tag", catalog.TpText, "")
	series.DefineAtrib("value", catalog.TpInt, 0)
	series.DefineAtrib("format", catalog.TpText, "")
	series.DefinePrimaryKeys("tag")
	if err := series.Init(); err != nil {
		return err
	}

	return nil
}

/**
* CreateSerie: Creates a new serie
* @param tag, format string, value int
* @return error
**/
func CreateSerie(tag, format string, value int) error {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader, imLeader := syn.getLeader()
	if !imLeader {
		return syn.createSerie(leader, tag, format, value)
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
* SetSerie: Sets a serie
* @param tag string, value int
* @return error
**/
func SetSerie(tag string, value int) error {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader, imLeader := syn.getLeader()
	if !imLeader {
		return syn.setSerie(leader, tag, value)
	}

	err := initSeries()
	if err != nil {
		return err
	}

	_, err = series.
		Update(et.Json{
			"value": value,
		}).
		Where(catalog.Eq("tag", tag)).
		Execute(nil)
	return err
}

/**
* GetSerie: Gets a serie
* @param tag string
* @return et.Json, error
**/
func GetSerie(tag string) (et.Json, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader, imLeader := syn.getLeader()
	if !imLeader {
		return syn.getSerie(leader, tag)
	}

	err := initSeries()
	if err != nil {
		return nil, err
	}

	items, err := series.
		Update(et.Json{}).
		BeforeUpdateFn(func(tx *catalog.Tx, old, new et.Json) error {
			value := old.Int("value")
			new["value"] = value + 1
			return nil
		}).
		Where(catalog.Eq("tag", tag)).
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

/**
* DropSerie: Drops a serie
* @param tag string
* @return error
**/
func DropSerie(tag string) error {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "tag")
	}

	leader, imLeader := syn.getLeader()
	if !imLeader {
		return syn.dropSerie(leader, tag)
	}

	err := initSeries()
	if err != nil {
		return err
	}

	_, err = series.
		Delete().
		Where(catalog.Eq("tag", tag)).
		Execute(nil)
	return err
}

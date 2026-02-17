package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

var series *catalog.Model

/**
* initSeries: Initializes the series model
* @param db *DB
* @return error
**/
func (s *Node) initSeries() error {
	if series != nil {
		return nil
	}

	db, err := s.coreDb()
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
func (s *Node) CreateSerie(tag, format string, value int) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.CreateSerie(tag, format, value)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.CreateSerie", tag, format, value)
		if res.Error != nil {
			return res.Error
		}
		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* SetSerie: Sets a serie
* @param tag string, value int
* @return error
**/
func (s *Node) SetSerie(tag string, value int) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.SetSerie(tag, value)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.SetSerie", tag, value)
		if res.Error != nil {
			return res.Error
		}
		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* GetSerie: Gets a serie
* @param tag string
* @return et.Item, error
**/
func (s *Node) GetSerie(tag string) (et.Item, error) {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.GetSerie(tag)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.GetSerie", tag)
		if res.Error != nil {
			return et.Item{}, res.Error
		}

		var result et.Item
		err := res.Get(&result)
		if err != nil {
			return et.Item{}, err
		}

		return result, nil
	}

	return et.Item{}, errors.New(msg.MSG_LEADER_NOT_FOUND)
}

/**
* DropSerie: Drops a serie
* @param tag string
* @return error
**/
func (s *Node) DropSerie(tag string) error {
	leader, imLeader := s.GetLeader()
	if imLeader {
		return s.lead.DropSerie(tag)
	}

	if leader != nil {
		res := s.Request(leader, "Leader.DropSerie", tag)
		if res.Error != nil {
			return res.Error
		}

		var result et.Item
		err := res.Get(&result)
		if err != nil {
			return err
		}

		return nil
	}

	return errors.New(msg.MSG_LEADER_NOT_FOUND)
}

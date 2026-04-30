package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Command struct {
	model        *Model
	command      Cmd
	tx           *Tx
	idx          string
	data         et.Json
	BeforeInsert func(model *Model, old, new et.Join) error
	BeforeUpdate func(model *Model, old, new et.Join) error
	BeforeDelete func(model *Model, old, new et.Join) error
	AfterInsert  func(model *Model, old, new et.Join) error
	AfterUpdate  func(model *Model, old, new et.Join) error
	AfterDelete  func(model *Model, old, new et.Join) error
}

func (c *Command) Execute() (et.Item, error) {
	switch c.command {
	case INSERT:
		tx, err := c.model.Insert(c.data, c.tx)
		if err != nil {
			return et.Item{}, err
		}
		result := tx.Result()
		if len(result) > 0 {
			return et.NewItem(result[0]), nil
		}
		return et.Item{}, nil
	case UPDATE:
		tx, err := c.model.Update(c.data, c.tx)
		if err != nil {
			return et.Item{}, err
		}
		result := tx.Result()
		if len(result) > 0 {
			return et.NewItem(result[0]), nil
		}
		return et.Item{}, nil
	case DELETE:
		tx, err := c.model.Delete(c.idx, c.tx)
		if err != nil {
			return et.Item{}, err
		}
		result := tx.Result()
		if len(result) > 0 {
			return et.NewItem(result[0]), nil
		}
		return et.Item{}, nil
	default:
		return et.Item{}, errors.New(msg.MSG_COMMAND_NOT_FOUND)
	}
}

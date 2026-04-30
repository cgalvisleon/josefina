package jdb

import "github.com/cgalvisleon/et/et"

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

func (c *Command) Execute() error {
	switch c.command {
	case INSERT:
		_, err := c.model.Insert(c.data, c.tx)
		return err
	case UPDATE:
		_, err := c.model.Update(c.data, c.tx)
		return err
	case DELETE:
		_, err := c.model.Delete(c.idx, c.tx)
		return err
	default:
		return nil
	}
}

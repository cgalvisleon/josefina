package jdb

import (
	"github.com/cgalvisleon/et/et"
)

type Command struct {
	model        *Model
	command      Cmd
	tx           *Tx
	idx          string
	data         []et.Json
	where        *Where
	beforeInsert []func(model *Model, old, new et.Join) error
	beforeUpdate []func(model *Model, old, new et.Join) error
	beforeDelete []func(model *Model, old, new et.Join) error
	afterInsert  []func(model *Model, old, new et.Join) error
	afterUpdate  []func(model *Model, old, new et.Join) error
	afterDelete  []func(model *Model, old, new et.Join) error
}

/**
* newCommand
* @param model *Model, cmd Cmd, idx string, data et.Json
* @return *Command
**/
func newCommand(model *Model, cmd Cmd, idx string, data []et.Json) *Command {
	return &Command{
		model:        model,
		command:      cmd,
		idx:          idx,
		data:         data,
		where:        nil,
		beforeInsert: []func(model *Model, old, new et.Join) error{},
		beforeUpdate: []func(model *Model, old, new et.Join) error{},
		beforeDelete: []func(model *Model, old, new et.Join) error{},
		afterInsert:  []func(model *Model, old, new et.Join) error{},
		afterUpdate:  []func(model *Model, old, new et.Join) error{},
		afterDelete:  []func(model *Model, old, new et.Join) error{},
	}
}

/**
* Where
* @param condition *et.Condition
* @return *Where
**/
func (s *Command) Where(condition *et.Condition) *Where {
	s.where = newWhere(s.model)
	s.where.Add(condition)
	return s.where
}

/**
* Exec
* @return (et.Item, error)
**/
func (s *Command) Exec() (et.Items, error) {
	switch s.command {
	case INSERT:
		return s.insertCmd()
	case UPDATE:
		return s.updateCmd()
	case DELETE:
		return s.deleteCmd()
	case UPSERT:
		return s.upsertCmd()
	case BULK:
		return s.bulkCmd()
	}
	return et.Items{}, nil
}

/**
* One
* @return (et.Item, error)
**/
func (s Command) One() (et.Item, error) {
	items, err := s.Exec()
	if err != nil {
		return et.Item{}, err
	}
	return items.First()
}

/**
* insertCmd
* @return et.Items, error
**/
func (s *Command) insertCmd() (et.Items, error) {
	return et.Items{}, nil
}

/**
* updateCmd
* @return et.Items, error
**/
func (s *Command) updateCmd() (et.Items, error) {
	return et.Items{}, nil
}

/**
* deleteCmd
* @return et.Items, error
**/
func (s *Command) deleteCmd() (et.Items, error) {
	return et.Items{}, nil
}

/**
* upsertCmd
* @return et.Items, error
**/
func (s *Command) upsertCmd() (et.Items, error) {
	return et.Items{}, nil
}

/**
* bulkCmd
* @return et.Items, error
**/
func (s *Command) bulkCmd() (et.Items, error) {
	return et.Items{}, nil
}

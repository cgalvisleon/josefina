package jdb

import (
	"github.com/cgalvisleon/et/et"
)

type Command struct {
	model        *Model
	command      Cmd
	tx           *Tx
	idx          string
	data         et.Json
	where        *Where
	beforeInsert []func(model *Model, old, new et.Join) error
	beforeUpdate []func(model *Model, old, new et.Join) error
	beforeDelete []func(model *Model, old, new et.Join) error
	afterInsert  []func(model *Model, old, new et.Join) error
	afterUpdate  []func(model *Model, old, new et.Join) error
	afterDelete  []func(model *Model, old, new et.Join) error
}

func (s *Command) Where(condition *et.Condition) *Where {
	s.where = newWhere(s.model)
	s.where.Add(condition)
	return s.where
}

/**
* Exec
* @return (et.Item, error)
**/
func (c *Command) Exec() (et.Item, error) {
	return et.Item{}, nil
}

/**
* newCommand
* @param model *Model, cmd Cmd, idx string, data et.Json
* @return *Command
**/
func newCommand(model *Model, cmd Cmd, idx string, data et.Json) *Command {
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
* insertCmd
* @param model *Model, data et.Json
* @return *Command
**/
func insertCmd(model *Model, data et.Json) *Command {
	idx := data.Str(INDEX)
	result := newCommand(model, INSERT, idx, data)
	return result
}

/**
* updateCmd
* @param model *Model, data et.Json
* @return *Command
**/
func updateCmd(model *Model, data et.Json) *Command {
	idx := data.Str(INDEX)
	result := newCommand(model, UPDATE, idx, data)
	return result
}

/**
* deleteCmd
* @param model *Model, idx string
* @return *Command
**/
func deleteCmd(model *Model, idx string) *Command {
	result := newCommand(model, DELETE, idx, et.Json{})
	return result
}

/**
* upsertCmd
* @param model *Model, data et.Json
* @return *Command
**/
func upsertCmd(model *Model, data et.Json) *Command {
	idx := data.Str(INDEX)
	result := newCommand(model, UPSERT, idx, data)
	return result
}

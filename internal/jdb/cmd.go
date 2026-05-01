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
	items        []et.Json
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
* @param model *Model, cmd Cmd, idx string, items et.Json
* @return *Command
**/
func newCommand(model *Model, cmd Cmd, idx string, items []et.Json) *Command {
	return &Command{
		model:        model,
		command:      cmd,
		idx:          idx,
		items:        items,
		where:        newWhere(model),
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
func (s *Command) Where(condition *et.Condition) *Command {
	s.where = newWhere(s.model)
	s.where.Add(condition)
	return s
}

/**
* And
* @param condition *et.Condition
* @return *Command
**/
func (s *Command) And(condition *et.Condition) *Command {
	if s.where == nil {
		s.where = newWhere(s.model)
	}
	s.where.And(condition)
	return s
}

/**
* Or
* @param condition *et.Condition
* @return *Command
**/
func (s *Command) Or(condition *et.Condition) *Command {
	if s.where == nil {
		s.where = newWhere(s.model)
	}
	s.where.Or(condition)
	return s
}

/**
* ExecTx
* @param tx *Tx
* @return (et.Item, error)
**/
func (s *Command) ExecTx(tx *Tx) (et.Items, error) {
	switch s.command {
	case INSERT:
		return s.insertCmd(tx)
	case UPDATE:
		return s.updateCmd(tx)
	case DELETE:
		return s.deleteCmd(tx)
	case UPSERT:
		return s.upsertCmd(tx)
	case BULK:
		return s.bulkCmd(tx)
	}
	return et.Items{}, nil
}

/**
* Exec
* @return (et.Items, error)
**/
func (s *Command) Exec() (et.Items, error) {
	return s.ExecTx(nil)
}

/**
* OneTx
* @param tx *Tx
* @return (et.Item, error)
**/
func (s *Command) OneTx(tx *Tx) (et.Item, error) {
	items, err := s.ExecTx(tx)
	if err != nil {
		return et.Item{}, err
	}
	return items.First()
}

/**
* One
* @return (et.Item, error)
**/
func (s *Command) One() (et.Item, error) {
	return s.OneTx(nil)
}

/**
* insertCmd
* @return et.Items, error
**/
func (s *Command) insertCmd(tx *Tx) (et.Items, error) {
	result := et.Items{}
	execute := tx == nil
	tx = GetTx(s.model.db, tx)
	for _, item := range s.items {
		idx := item.Str(INDEX)
		err := s.model.insert(idx, item, tx)
		if err != nil {
			return et.Items{}, err
		}

		if execute {
			_, err = tx.Commit()
			if err != nil {
				return et.Items{}, err
			}
		}
		result.Add(tx.Result)
	}
	return result, nil
}

/**
* updateCmd
* @return et.Items, error
**/
func (s *Command) updateCmd(tx *Tx) (et.Items, error) {
	result := et.Items{}
	execute := tx == nil
	tx = GetTx(s.model.db, tx)
	items, err := s.where.All(tx)
	if err != nil {
		return et.Items{}, err
	}
	for _, item := range items.Result {
		idx := item.Str(INDEX)
		err := s.model.update(idx, item, tx)
		if err != nil {
			return et.Items{}, err
		}

		if execute {
			_, err = tx.Commit()
			if err != nil {
				return et.Items{}, err
			}
		}
		result.Add(tx.Result)
	}
	return result, nil
}

/**
* deleteCmd
* @return et.Items, error
**/
func (s *Command) deleteCmd(tx *Tx) (et.Items, error) {
	result := et.Items{}
	execute := tx == nil
	tx = GetTx(s.model.db, tx)
	if len(s.where.conditions) == 0 {
		return result, errors.New(msg.MSG_NO_CONDITIONS)
	}
	items, err := s.where.All(tx)
	if err != nil {
		return et.Items{}, err
	}
	for _, item := range items.Result {
		idx := item.Str(INDEX)
		err := s.model.delete(idx, tx)
		if err != nil {
			return et.Items{}, err
		}

		if execute {
			_, err = tx.Commit()
			if err != nil {
				return et.Items{}, err
			}
		}
		result.Add(tx.Result)
	}
	return result, nil
}

/**
* upsertCmd
* @return et.Items, error
**/
func (s *Command) upsertCmd(tx *Tx) (et.Items, error) {
	result := et.Items{}
	execute := tx == nil
	tx = GetTx(s.model.db, tx)
	for _, item := range s.items {
		idx := item.Str(INDEX)
		err := s.model.upsert(idx, item, tx)
		if err != nil {
			return et.Items{}, err
		}

		if execute {
			_, err = tx.Commit()
			if err != nil {
				return et.Items{}, err
			}
		}
		result.Add(tx.Result)
	}
	return result, nil
}

/**
* bulkCmd
* @return et.Items, error
**/
func (s *Command) bulkCmd(tx *Tx) (et.Items, error) {
	return s.insertCmd(tx)
}

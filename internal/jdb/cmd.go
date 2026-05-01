package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type FnTrigger func(model *Model, old, new *et.Json, tx *Tx) error

type Command struct {
	model        *Model
	command      Cmd
	tx           *Tx
	items        []et.Json
	where        *Where
	beforeInsert []FnTrigger
	beforeUpdate []FnTrigger
	beforeDelete []FnTrigger
	afterInsert  []FnTrigger
	afterUpdate  []FnTrigger
	afterDelete  []FnTrigger
}

/**
* newCommand
* @param model *Model, cmd Cmd, idx string, items et.Json
* @return *Command
**/
func newCommand(model *Model, cmd Cmd, items []et.Json) *Command {
	return &Command{
		model:        model,
		command:      cmd,
		items:        items,
		where:        newWhere(model),
		beforeInsert: make([]FnTrigger, 0),
		beforeUpdate: make([]FnTrigger, 0),
		beforeDelete: make([]FnTrigger, 0),
		afterInsert:  make([]FnTrigger, 0),
		afterUpdate:  make([]FnTrigger, 0),
		afterDelete:  make([]FnTrigger, 0),
	}
}

/**
* Add
* @param condition *et.Condition
* @return *Command
**/
func (s *Command) Add(condition *et.Condition) *Command {
	s.where.Add(condition)
	return s
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
		return s.insertCmd(tx)
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
	tx.beforeInsert = s.beforeInsert
	tx.beforeUpdate = s.beforeUpdate
	tx.beforeDelete = s.beforeDelete
	tx.afterInsert = s.afterInsert
	tx.afterUpdate = s.afterUpdate
	tx.afterDelete = s.afterDelete
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
	tx.beforeInsert = s.beforeInsert
	tx.beforeUpdate = s.beforeUpdate
	tx.beforeDelete = s.beforeDelete
	tx.afterInsert = s.afterInsert
	tx.afterUpdate = s.afterUpdate
	tx.afterDelete = s.afterDelete
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
	tx.beforeInsert = s.beforeInsert
	tx.beforeUpdate = s.beforeUpdate
	tx.beforeDelete = s.beforeDelete
	tx.afterInsert = s.afterInsert
	tx.afterUpdate = s.afterUpdate
	tx.afterDelete = s.afterDelete
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
	tx.beforeInsert = s.beforeInsert
	tx.beforeUpdate = s.beforeUpdate
	tx.beforeDelete = s.beforeDelete
	tx.afterInsert = s.afterInsert
	tx.afterUpdate = s.afterUpdate
	tx.afterDelete = s.afterDelete
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
* BeforeInserts
* @param fn FnTrigger
* @return *Command
**/
func (s *Command) BeforeInserts(fn FnTrigger) *Command {
	s.beforeInsert = append(s.beforeInsert, fn)
	return s
}

/**
* BeforeUpdates
* @param fn FnTrigger
* @return *Command
**/
func (s *Command) BeforeUpdates(fn FnTrigger) *Command {
	s.beforeUpdate = append(s.beforeUpdate, fn)
	return s
}

/**
* BeforeDeletes
* @param fn FnTrigger
* @return *Command
**/
func (s *Command) BeforeDeletes(fn FnTrigger) *Command {
	s.beforeDelete = append(s.beforeDelete, fn)
	return s
}

/**
* AfterInserts
* @param fn FnTrigger
* @return *Command
**/
func (s *Command) AfterInserts(fn FnTrigger) *Command {
	s.afterInsert = append(s.afterInsert, fn)
	return s
}

/**
* AfterUpdates
* @param fn FnTrigger
* @return *Command
**/
func (s *Command) AfterUpdates(fn FnTrigger) *Command {
	s.afterUpdate = append(s.afterUpdate, fn)
	return s
}

/**
* AfterDeletes
* @param fn FnTrigger
* @return *Command
**/
func (s *Command) AfterDeletes(fn FnTrigger) *Command {
	s.afterDelete = append(s.afterDelete, fn)
	return s
}

type Diud struct {
	Schema string         `json:"schema"`
	Name   string         `json:"name"`
	Data   et.Json        `json:"data"`
	Where  []et.Condition `json:"where"`
}

type DBulk struct {
	Schema string    `json:"schema"`
	Name   string    `json:"name"`
	Data   []et.Json `json:"data"`
}

type DCmd struct {
	Insert *Diud  `json:"insert"`
	Update *Diud  `json:"update"`
	Delete *Diud  `json:"delete"`
	Bulk   *DBulk `json:"bulk"`
}

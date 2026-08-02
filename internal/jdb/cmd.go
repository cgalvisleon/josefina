package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* TriggerFn: Callback signature for before/after insert, update, and delete hooks.
**/
type TriggerFn func(model *Model, old, new *et.Json, tx *Tx) error

/**
* Command: Builder for DML operations; accumulates items, conditions, and trigger callbacks
* before execution via Exec or ExecTx.
**/
type Command struct {
	command      Cmd
	model        *Model
	items        []et.Json
	where        *Where
	tx           *Tx
	beforeInsert []TriggerFn
	beforeUpdate []TriggerFn
	beforeDelete []TriggerFn
	afterInsert  []TriggerFn
	afterUpdate  []TriggerFn
	afterDelete  []TriggerFn
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
		beforeInsert: make([]TriggerFn, 0),
		beforeUpdate: make([]TriggerFn, 0),
		beforeDelete: make([]TriggerFn, 0),
		afterInsert:  make([]TriggerFn, 0),
		afterUpdate:  make([]TriggerFn, 0),
		afterDelete:  make([]TriggerFn, 0),
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
	var isCommitted bool
	tx, isCommitted = GetTx(s.model.db, tx)
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
		result.Add(tx.Result)
	}
	if isCommitted {
		_, err := tx.Commit()
		if err != nil {
			return et.Items{}, err
		}
	}
	return result, nil
}

/**
* updateCmd
* @return et.Items, error
**/
func (s *Command) updateCmd(tx *Tx) (et.Items, error) {
	result := et.Items{}
	var isCommitted bool
	tx, isCommitted = GetTx(s.model.db, tx)
	tx.beforeInsert = s.beforeInsert
	tx.beforeUpdate = s.beforeUpdate
	tx.beforeDelete = s.beforeDelete
	tx.afterInsert = s.afterInsert
	tx.afterUpdate = s.afterUpdate
	tx.afterDelete = s.afterDelete
	s.where.keepIdx = true
	items, err := s.where.AllTx(tx)
	if err != nil {
		return et.Items{}, err
	}
	newData := et.Json{}
	if len(s.items) > 0 {
		newData = s.items[0]
	}
	for _, item := range items.Result {
		idx := item.Str(INDEX)
		merged := make(et.Json, len(item)+len(newData))
		for k, v := range item {
			merged[k] = v
		}
		for k, v := range newData {
			merged[k] = v
		}
		delete(merged, INDEX)
		err := s.model.update(idx, merged, tx)
		if err != nil {
			return et.Items{}, err
		}
		result.Add(tx.Result)
	}
	if isCommitted {
		_, err = tx.Commit()
		if err != nil {
			return et.Items{}, err
		}
	}
	return result, nil
}

/**
* deleteCmd
* @return et.Items, error
**/
func (s *Command) deleteCmd(tx *Tx) (et.Items, error) {
	result := et.Items{}
	var isCommitted bool
	tx, isCommitted = GetTx(s.model.db, tx)
	tx.beforeInsert = s.beforeInsert
	tx.beforeUpdate = s.beforeUpdate
	tx.beforeDelete = s.beforeDelete
	tx.afterInsert = s.afterInsert
	tx.afterUpdate = s.afterUpdate
	tx.afterDelete = s.afterDelete
	if len(s.where.conditions) == 0 {
		return result, errors.New(msg.MSG_NO_CONDITIONS)
	}
	s.where.keepIdx = true
	items, err := s.where.AllTx(tx)
	if err != nil {
		return et.Items{}, err
	}
	for _, item := range items.Result {
		idx := item.Str(INDEX)
		err := s.model.delete(idx, tx)
		if err != nil {
			return et.Items{}, err
		}
		result.Add(tx.Result)
	}

	if isCommitted {
		_, err = tx.Commit()
		if err != nil {
			return et.Items{}, err
		}
	}
	return result, nil
}

/**
* upsertCmd
* @return et.Items, error
**/
func (s *Command) upsertCmd(tx *Tx) (et.Items, error) {
	result := et.Items{}
	var isCommitted bool
	tx, isCommitted = GetTx(s.model.db, tx)
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
		result.Add(tx.Result)
	}
	if isCommitted {
		_, err := tx.Commit()
		if err != nil {
			return et.Items{}, err
		}
	}
	return result, nil
}

/**
* BeforeInserts
* @param fn TriggerFn
* @return *Command
**/
func (s *Command) BeforeInserts(fn TriggerFn) *Command {
	s.beforeInsert = append(s.beforeInsert, fn)
	return s
}

/**
* BeforeUpdates
* @param fn TriggerFn
* @return *Command
**/
func (s *Command) BeforeUpdates(fn TriggerFn) *Command {
	s.beforeUpdate = append(s.beforeUpdate, fn)
	return s
}

/**
* BeforeDeletes
* @param fn TriggerFn
* @return *Command
**/
func (s *Command) BeforeDeletes(fn TriggerFn) *Command {
	s.beforeDelete = append(s.beforeDelete, fn)
	return s
}

/**
* AfterInserts
* @param fn TriggerFn
* @return *Command
**/
func (s *Command) AfterInserts(fn TriggerFn) *Command {
	s.afterInsert = append(s.afterInsert, fn)
	return s
}

/**
* AfterUpdates
* @param fn TriggerFn
* @return *Command
**/
func (s *Command) AfterUpdates(fn TriggerFn) *Command {
	s.afterUpdate = append(s.afterUpdate, fn)
	return s
}

/**
* AfterDeletes
* @param fn TriggerFn
* @return *Command
**/
func (s *Command) AfterDeletes(fn TriggerFn) *Command {
	s.afterDelete = append(s.afterDelete, fn)
	return s
}

/**
* Diud: Wire format for a single insert, update, or delete command sent over the API.
**/
type Diud struct {
	Schema string         `json:"schema"`
	Name   string         `json:"name"`
	Items  []et.Json      `json:"items"`
	Where  []et.Condition `json:"where"`
}

/**
* First: Returns the first item
* @return et.Json
**/
func (s *Diud) First() et.Json {
	if len(s.Items) == 0 {
		return et.Json{}
	}
	return s.Items[0]
}

/**
* DCmd: Envelope that carries exactly one of insert, update, delete, or bulk in a command batch.
**/
type DCmd struct {
	Insert *Diud `json:"insert"`
	Update *Diud `json:"update"`
	Delete *Diud `json:"delete"`
	Bulk   *Diud `json:"bulk"`
}

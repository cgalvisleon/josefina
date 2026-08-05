package jdb

import (
	"github.com/cgalvisleon/et/et"
)

/**
* Cmd: Identifies the DML operation type for a command or transaction entry.
**/
type Cmd string

const (
	QUERY  Cmd = "query"
	INSERT Cmd = "insert"
	UPDATE Cmd = "update"
	DELETE Cmd = "delete"
	UPSERT Cmd = "upsert"
	BULK   Cmd = "bulk"
)

/**
* Command: Builder for DML operations; accumulates items, conditions, and trigger callbacks
* before execution via Exec or ExecTx.
**/
type Command struct {
	command       Cmd             `json:"-"`
	model         *Model          `json:"-"`
	items         []et.Json       `json:"-"`
	wheres        []*et.Condition `json:"-"`
	results       []string        `json:"-"`
	beforeInserts []*Trigger      `json:"-"`
	afterInserts  []*Trigger      `json:"-"`
	beforeUpdates []*Trigger      `json:"-"`
	afterUpdates  []*Trigger      `json:"-"`
	beforeDeletes []*Trigger      `json:"-"`
	afterDeletes  []*Trigger      `json:"-"`
}

/**
* newCommand
* @param model *Model, cmd Cmd, idx string, items et.Json
* @return *Command
**/
func newCommand(model *Model, cmd Cmd, items []et.Json) *Command {
	result := &Command{
		command:       cmd,
		model:         model,
		items:         items,
		wheres:        make([]*et.Condition, 0),
		results:       make([]string, 0),
		beforeInserts: make([]*Trigger, 0),
		afterInserts:  make([]*Trigger, 0),
		beforeUpdates: make([]*Trigger, 0),
		afterUpdates:  make([]*Trigger, 0),
		beforeDeletes: make([]*Trigger, 0),
		afterDeletes:  make([]*Trigger, 0),
	}
	for _, trigger := range model.BeforeInserts {
		result.beforeInserts = append(result.beforeInserts, trigger)
	}
	for _, trigger := range model.AfterInserts {
		result.afterInserts = append(result.afterInserts, trigger)
	}
	for _, trigger := range model.BeforeUpdates {
		result.beforeUpdates = append(result.beforeUpdates, trigger)
	}
	for _, trigger := range model.AfterUpdates {
		result.afterUpdates = append(result.afterUpdates, trigger)
	}
	for _, trigger := range model.BeforeDeletes {
		result.beforeDeletes = append(result.beforeDeletes, trigger)
	}
	for _, trigger := range model.AfterDeletes {
		result.afterDeletes = append(result.afterDeletes, trigger)
	}
	return result
}

/**
* Add
* @param condition *et.Condition
* @return *Command
**/
func (s *Command) Add(condition *et.Condition) *Command {
	if len(s.wheres) > 0 && condition.Connector == et.NaC {
		condition.Connector = et.And
	}

	s.wheres = append(s.wheres, condition)
	return s
}

/**
* Where
* @param condition *et.Condition
* @return *Query
**/
func (s *Command) Where(condition *et.Condition) *Command {
	return s.Add(condition)
}

/**
* And
* @param condition *et.Condition
* @return *Query
**/
func (s *Command) And(condition *et.Condition) *Command {
	condition.Connector = et.And
	return s.Add(condition)
}

/**
* Or
* @param condition *et.Condition
* @return *Query
**/
func (s *Command) Or(condition *et.Condition) *Command {
	condition.Connector = et.Or
	return s.Add(condition)
}

/**
* BeforeInserts
* @param trigger *Trigger
* @return *Command
**/
func (s *Command) BeforeInserts(trigger *Trigger) *Command {
	s.beforeInserts = append(s.beforeInserts, trigger)
	return s
}

/**
* BeforeUpdates
* @param trigger *Trigger
* @return *Command
**/
func (s *Command) BeforeUpdates(trigger *Trigger) *Command {
	s.beforeUpdates = append(s.beforeUpdates, trigger)
	return s
}

/**
* BeforeDeletes
* @param trigger *Trigger
* @return *Command
**/
func (s *Command) BeforeDeletes(trigger *Trigger) *Command {
	s.beforeDeletes = append(s.beforeDeletes, trigger)
	return s
}

/**
* AfterInserts
* @param trigger *Trigger
* @return *Command
**/
func (s *Command) AfterInserts(trigger *Trigger) *Command {
	s.afterInserts = append(s.afterInserts, trigger)
	return s
}

/**
* AfterUpdates
* @param trigger *Trigger
* @return *Command
**/
func (s *Command) AfterUpdates(trigger *Trigger) *Command {
	s.afterUpdates = append(s.afterUpdates, trigger)
	return s
}

/**
* AfterDeletes
* @param fn Trigger
* @return *Command
**/
func (s *Command) AfterDeletes(trigger *Trigger) *Command {
	s.afterDeletes = append(s.afterDeletes, trigger)
	return s
}

/**
* Exec
* @return []et.Json, error
**/
func (s *Command) Exec() ([]et.Json, error) {
	result := []et.Json{}
	return result, nil
}

/**
* One
* @return (et.Item, error)
**/
func (s *Command) One() (et.Json, error) {
	result, err := s.Exec()
	if err != nil {
		return et.Json{}, err
	}

	if len(result) == 0 {
		return et.Json{}, nil
	}

	return result[0], nil
}

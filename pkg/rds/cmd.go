package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

type Command string

const (
	INSERT Command = "insert"
	UPDATE Command = "update"
	DELETE Command = "delete"
)

type Cmd struct {
	db                   *DB               `json:"-"`
	model                *Model            `json:"-"`
	command              Command           `json:"-"`
	beforeTriggerInserts []*Trigger        `json:"-"`
	beforeTriggerUpdates []*Trigger        `json:"-"`
	beforeTriggerDeletes []*Trigger        `json:"-"`
	afterTriggerInserts  []*Trigger        `json:"-"`
	afterTriggerUpdates  []*Trigger        `json:"-"`
	afterTriggerDeletes  []*Trigger        `json:"-"`
	beforeInserts        []TriggerFunction `json:"-"`
	afterInserts         []TriggerFunction `json:"-"`
	beforeUpdates        []TriggerFunction `json:"-"`
	afterUpdates         []TriggerFunction `json:"-"`
	beforeDeletes        []TriggerFunction `json:"-"`
	afterDeletes         []TriggerFunction `json:"-"`
}

/**
* newCmd: Creates a new command
* @param model *Model, command Command
* @return *Cmd
**/
func newCmd(model *Model) *Cmd {
	result := &Cmd{
		db:                   model.db,
		model:                model,
		beforeTriggerInserts: make([]*Trigger, 0),
		beforeTriggerUpdates: make([]*Trigger, 0),
		beforeTriggerDeletes: make([]*Trigger, 0),
		afterTriggerInserts:  make([]*Trigger, 0),
		afterTriggerUpdates:  make([]*Trigger, 0),
		afterTriggerDeletes:  make([]*Trigger, 0),
		beforeInserts:        make([]TriggerFunction, 0),
		afterInserts:         make([]TriggerFunction, 0),
		beforeUpdates:        make([]TriggerFunction, 0),
		afterUpdates:         make([]TriggerFunction, 0),
		beforeDeletes:        make([]TriggerFunction, 0),
		afterDeletes:         make([]TriggerFunction, 0),
	}
	for _, trigger := range model.BeforeInserts {
		result.beforeTriggerInserts = append(result.beforeTriggerInserts, trigger)
	}
	for _, trigger := range model.AfterInserts {
		result.afterTriggerInserts = append(result.afterTriggerInserts, trigger)
	}
	for _, trigger := range model.BeforeUpdates {
		result.beforeTriggerUpdates = append(result.beforeTriggerUpdates, trigger)
	}
	for _, trigger := range model.AfterUpdates {
		result.afterTriggerUpdates = append(result.afterTriggerUpdates, trigger)
	}
	for _, trigger := range model.BeforeDeletes {
		result.beforeTriggerDeletes = append(result.beforeTriggerDeletes, trigger)
	}
	for _, trigger := range model.AfterDeletes {
		result.afterTriggerDeletes = append(result.afterTriggerDeletes, trigger)
	}

	return result
}

/**
* runTrigger
* @param trigger *Trigger, tx *Tx, old et.Json, new et.Json
* @return error
**/
func (s *Cmd) runTrigger(trigger *Trigger, tx *Tx, old, new et.Json) error {
	model := s.model
	vm, ok := model.triggers[trigger.Name]
	if !ok {
		vm = newVm()
		model.triggers[trigger.Name] = vm
	}

	vm.Set("self", model)
	vm.Set("tx", tx)
	vm.Set("old", old)
	vm.Set("new", new)
	script := string(trigger.Definition)
	_, err := vm.Run(script)
	if err != nil {
		return err
	}

	return nil
}

/**
* beforeInsert
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) beforeInsert(fn TriggerFunction) *Cmd {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* afterInsert
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) afterInsert(fn TriggerFunction) *Cmd {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* beforeUpdate
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) beforeUpdate(fn TriggerFunction) *Cmd {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* afterUpdate
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) afterUpdate(fn TriggerFunction) *Cmd {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* beforeDelete
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) beforeDelete(fn TriggerFunction) *Cmd {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* afterDelete
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) afterDelete(fn TriggerFunction) *Cmd {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* insert: Inserts the model
* @param tx *Tx, new et.Json
* @return et.Json, error
**/
func (s *Cmd) insert(tx *Tx, new et.Json) (et.Json, error) {
	s.command = INSERT
	tx, commit := getTx(tx)
	model := s.model

	// Validate required fields
	for _, name := range model.Required {
		if _, ok := new[name]; !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
	}

	// Validate unique fields
	for _, name := range model.Unique {
		if _, ok := new[name]; !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
		source := model.data[name]
		key := fmt.Sprintf("%v", new[name])
		if source.IsExist(key) {
			return nil, fmt.Errorf(msg.MSG_RECORD_EXISTS)
		}
	}

	for name, detail := range model.References {
		if _, ok := new[name]; !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}

		to, err := s.db.getModel(detail.To.Schema, detail.To.Name)
		if err != nil {
			return nil, err
		}

		fk, ok := detail.Keys[name]
		if !ok {
			return nil, fmt.Errorf(msg.MSG_FOREIGN_KEY_NOT_FOUND, name)
		}

		key := fmt.Sprintf("%v", new[name])
		ok, err = to.isExisted(fk, key)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, fmt.Errorf(msg.MSG_VIOLATE_FOREIGN_KEY, name)
		}
	}

	idx := new.ValStr("", INDEX)
	if idx == "" {
		idx = model.getJid()
		new[INDEX] = idx
	}

	// Run before insert triggers
	for _, trigger := range s.beforeTriggerInserts {
		err := s.runTrigger(trigger, tx, et.Json{}, new)
		if err != nil {
			return nil, err
		}
	}

	// Run before insert trigger function
	for _, fn := range s.beforeInserts {
		err := fn(tx, et.Json{}, new)
		if err != nil {
			return nil, err
		}
	}

	// Insert data into indexes
	tx.add(model, INSERT, idx, new)

	// Run after insert triggers
	for _, trigger := range s.afterTriggerInserts {
		err := s.runTrigger(trigger, tx, et.Json{}, new)
		if err != nil {
			return nil, err
		}
	}

	// Run after insert trigger function
	for _, fn := range s.afterInserts {
		err := fn(tx, et.Json{}, new)
		if err != nil {
			return nil, err
		}
	}

	if commit {
		err := tx.commit()
		if err != nil {
			return nil, err
		}
	}

	return new, nil
}

/**
* update: Updates the model
* @param tx *Tx, data et.Json, where *Wheres
* @return []et.Json, error
**/
func (s *Cmd) update(tx *Tx, data et.Json, wheres *Wheres) ([]et.Json, error) {
	s.command = UPDATE
	tx, commit := getTx(tx)
	model := s.model
	wheres.SetOwner(model)
	items, err := wheres.Rows(tx)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	if len(items) == 0 {
		return result, nil
	}

	add := func(item et.Json) {
		result = append(result, item)
	}

	for _, old := range items {
		// Get index
		idx := old.ValStr("", INDEX)
		if idx == "" {
			return nil, errorRecordNotFound
		}

		// Update data
		new := old.Clone()
		for k, v := range data {
			new[k] = v
		}

		// Run before update triggers
		for _, trigger := range s.beforeTriggerUpdates {
			err := s.runTrigger(trigger, tx, old, new)
			if err != nil {
				return nil, err
			}
		}

		// Run before update trigger function
		for _, fn := range s.beforeUpdates {
			err := fn(tx, old, new)
			if err != nil {
				return nil, err
			}
		}

		// Insert data into indexes
		tx.add(model, UPDATE, idx, new)

		// Run after update triggers
		for _, trigger := range s.afterTriggerUpdates {
			err := s.runTrigger(trigger, tx, old, new)
			if err != nil {
				return nil, err
			}
		}

		// Run after update trigger function
		for _, fn := range s.afterUpdates {
			err := fn(tx, old, new)
			if err != nil {
				return nil, err
			}
		}

		add(new)
	}

	if commit {
		err := tx.commit()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* delete: Deletes the model
* @param tx *Tx, where *Wheres
* @return []et.Json, error
**/
func (s *Cmd) delete(tx *Tx, where *Wheres) ([]et.Json, error) {
	s.command = DELETE
	tx, commit := getTx(tx)
	model := s.model
	where.SetOwner(model)
	items, err := where.Rows(tx)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	if len(items) == 0 {
		return result, nil
	}

	add := func(item et.Json) {
		result = append(result, item)
	}

	for _, old := range items {
		// Get index
		idx := old.ValStr("", INDEX)
		if idx == "" {
			return nil, errorRecordNotFound
		}

		// Run before delete triggers
		for _, trigger := range s.beforeTriggerDeletes {
			err := s.runTrigger(trigger, tx, old, et.Json{})
			if err != nil {
				return nil, err
			}
		}

		// Run before delete trigger function
		for _, fn := range s.beforeDeletes {
			err := fn(tx, old, et.Json{})
			if err != nil {
				return nil, err
			}
		}

		// Delete data from indexes
		tx.add(model, DELETE, idx, old)

		// Run after delete triggers
		for _, trigger := range s.afterTriggerDeletes {
			err := s.runTrigger(trigger, tx, old, et.Json{})
			if err != nil {
				return nil, err
			}
		}

		// Run after delete trigger function
		for _, fn := range s.afterDeletes {
			err := fn(tx, old, et.Json{})
			if err != nil {
				return nil, err
			}
		}

		add(old)
	}

	if commit {
		err := tx.commit()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* upsert: Upserts the model
* @param tx *Tx, new et.Json
* @return []et.Json, error
**/
func (s *Cmd) upsert(tx *Tx, new et.Json) ([]et.Json, error) {
	model := s.model
	where := newWhere()
	where.SetOwner(model)
	exists := true
	for _, name := range model.PrimaryKeys {
		source, ok := model.data[name]
		if !ok {
			return nil, errorPrimaryKeysNotFound
		}
		key := fmt.Sprintf("%v", new[name])
		if key == "" {
			return nil, errorPrimaryKeysNotFound
		}
		exists = source.IsExist(key)
		if !exists {
			break
		}
		where.Add(Eq(name, key))
	}

	if !exists {
		result, err := s.insert(tx, new)
		if err != nil {
			return nil, err
		}
		return []et.Json{result}, nil
	}

	return s.update(tx, new, where)
}

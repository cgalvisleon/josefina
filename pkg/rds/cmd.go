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
	UPSERT Command = "upsert"
)

type Cmd struct {
	db            *DB        `json:"-"`
	model         *Model     `json:"-"`
	command       Command    `json:"-"`
	beforeInserts []*Trigger `json:"-"`
	beforeUpdates []*Trigger `json:"-"`
	beforeDeletes []*Trigger `json:"-"`
	afterInserts  []*Trigger `json:"-"`
	afterUpdates  []*Trigger `json:"-"`
	afterDeletes  []*Trigger `json:"-"`
}

/**
* newCmd: Creates a new command
* @param model *Model, command Command
* @return *Cmd
**/
func newCmd(model *Model, command Command) *Cmd {
	result := &Cmd{
		db:            model.db,
		model:         model,
		command:       command,
		beforeInserts: make([]*Trigger, 0),
		beforeUpdates: make([]*Trigger, 0),
		beforeDeletes: make([]*Trigger, 0),
		afterInserts:  make([]*Trigger, 0),
		afterUpdates:  make([]*Trigger, 0),
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
* insert: Inserts the model
* @param tx *Tx, new et.Json
* @return et.Json, error
**/
func (s *Cmd) insert(tx *Tx, new et.Json) (et.Json, error) {
	tx, commit := getTx(tx)
	model := s.model
	idx := new.ValStr("", INDEX)
	if idx == "" {
		idx = model.getJid()
		new[INDEX] = idx
	}

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

	// Run before insert triggers
	for _, trigger := range s.beforeInserts {
		err := s.runTrigger(trigger, tx, et.Json{}, new)
		if err != nil {
			return nil, err
		}
	}

	// Insert data into indexes
	tx.add(model, INSERT, idx, new)

	// Run after insert triggers
	for _, trigger := range s.afterInserts {
		err := s.runTrigger(trigger, tx, et.Json{}, new)
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
func (s *Cmd) update(tx *Tx, data et.Json, where *Wheres) ([]et.Json, error) {
	tx, commit := getTx(tx)
	model := s.model
	items, err := where.Rows(tx)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
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
		for _, trigger := range s.beforeUpdates {
			err := s.runTrigger(trigger, tx, old, new)
			if err != nil {
				return nil, err
			}
		}

		// Insert data into indexes
		tx.add(model, UPDATE, idx, new)

		// Run after update triggers
		for _, trigger := range s.afterUpdates {
			err := s.runTrigger(trigger, tx, old, new)
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
	tx, commit := getTx(tx)
	model := s.model
	items, err := where.Rows(tx)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
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
		for _, trigger := range s.beforeDeletes {
			err := s.runTrigger(trigger, tx, old, et.Json{})
			if err != nil {
				return nil, err
			}
		}

		// Delete data from indexes
		tx.add(model, DELETE, idx, old)

		// Run after delete triggers
		for _, trigger := range s.afterDeletes {
			err := s.runTrigger(trigger, tx, old, et.Json{})
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
	where := newWhere(model)
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
		if !source.IsExist(key) {
			exists = false
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

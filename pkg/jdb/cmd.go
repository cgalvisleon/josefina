package jdb

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
	model                *Model            `json:"-"`
	data                 et.Json           `json:"-"`
	command              Command           `json:"-"`
	wheres               *Wheres           `json:"-"`
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
	isDebug              bool              `json:"-"`
}

/**
* newCmd: Creates a new command
* @param model *Model, command Command
* @return *Cmd
**/
func newCmd(model *Model) *Cmd {
	result := &Cmd{
		model:                model,
		data:                 et.Json{},
		wheres:               newWhere(),
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
* IsDebug: Returns the debug mode
* @return *Cmd
**/
func (s *Cmd) IsDebug() *Cmd {
	s.isDebug = true
	return s
}

/**
* BeforeInsertFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeInsertFn(fn TriggerFunction) *Cmd {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* AfterInsertFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterInsertFn(fn TriggerFunction) *Cmd {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* BeforeUpdateFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeUpdateFn(fn TriggerFunction) *Cmd {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* AfterUpdateFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterUpdateFn(fn TriggerFunction) *Cmd {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* BeforeDeleteFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) BeforeDeleteFn(fn TriggerFunction) *Cmd {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* AfterDeleteFn
* @param fn TriggerFunction
* @return *Cmd
**/
func (s *Cmd) AfterDeleteFn(fn TriggerFunction) *Cmd {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* BeforeInsert
* @param name, definition string
* @return *Cmd
**/
func (s *Cmd) BeforeInsert(name, definition string) *Cmd {
	definitionBytes := []byte(definition)
	s.beforeTriggerInserts = append(s.beforeTriggerInserts, &Trigger{
		Name:       name,
		Definition: definitionBytes,
	})
	return s
}

/**
* AfterInsert
* @param name, definition string
* @return *Cmd
**/
func (s *Cmd) AfterInsert(name, definition string) *Cmd {
	definitionBytes := []byte(definition)
	s.afterTriggerInserts = append(s.afterTriggerInserts, &Trigger{
		Name:       name,
		Definition: definitionBytes,
	})
	return s
}

/**
* BeforeUpdate
* @param name, definition string
* @return *Cmd
**/
func (s *Cmd) BeforeUpdate(name, definition string) *Cmd {
	definitionBytes := []byte(definition)
	s.beforeTriggerUpdates = append(s.beforeTriggerUpdates, &Trigger{
		Name:       name,
		Definition: definitionBytes,
	})
	return s
}

/**
* AfterUpdate
* @param name, definition string
* @return *Cmd
**/
func (s *Cmd) AfterUpdate(name, definition string) *Cmd {
	definitionBytes := []byte(definition)
	s.afterTriggerUpdates = append(s.afterTriggerUpdates, &Trigger{
		Name:       name,
		Definition: definitionBytes,
	})
	return s
}

/**
* BeforeDelete
* @param name, definition string
* @return *Cmd
**/
func (s *Cmd) BeforeDelete(name, definition string) *Cmd {
	definitionBytes := []byte(definition)
	s.beforeTriggerDeletes = append(s.beforeTriggerDeletes, &Trigger{
		Name:       name,
		Definition: definitionBytes,
	})
	return s
}

/**
* AfterDelete
* @param name, definition string
* @return *Cmd
**/
func (s *Cmd) AfterDelete(name, definition string) *Cmd {
	definitionBytes := []byte(definition)
	s.afterTriggerDeletes = append(s.afterTriggerDeletes, &Trigger{
		Name:       name,
		Definition: definitionBytes,
	})
	return s
}

/**
* executeInsert
* @param tx *Tx
* @return et.Json, error
**/
func (s *Cmd) executeInsert(tx *Tx) (et.Json, error) {
	model := s.model
	if model == nil {
		return nil, fmt.Errorf(msg.MSG_MODEL_IS_NIL)

	}

	if s.data.IsEmpty() {
		return nil, fmt.Errorf(msg.MSG_NOT_DATA)
	}

	new := s.data

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
		source, ok := model.stores[name]
		if !ok {
			return nil, fmt.Errorf(msg.MSG_STORE_NOT_FOUND, name)
		}
		key := fmt.Sprintf("%v", new[name])
		if source.IsExist(key) {
			return nil, fmt.Errorf(msg.MSG_RECORD_EXISTS)
		}
	}

	for name, detail := range model.References {
		if _, ok := new[name]; !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}

		fk, ok := detail.Keys[name]
		if !ok {
			return nil, fmt.Errorf(msg.MSG_FOREIGN_KEY_NOT_FOUND, name)
		}

		key := fmt.Sprintf("%v", new[name])
		exists, err := isExisted(detail.To, fk, key)
		if err != nil {
			return nil, err
		}

		if !exists {
			return nil, fmt.Errorf(msg.MSG_VIOLATE_FOREIGN_KEY, name)
		}
	}

	idx := new.ValStr("", INDEX)
	if idx == "" {
		idx = model.genKey()
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

	return et.Json{}, nil
}

/**
* executeUpdate
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) executeUpdate(tx *Tx) ([]et.Json, error) {
	model := s.model
	if model == nil {
		return nil, fmt.Errorf(msg.MSG_MODEL_IS_NIL)
	}

	s.wheres.SetOwner(model)
	items, err := s.wheres.Run(tx)
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	if len(items) == 0 {
		return result, nil
	}

	data := s.data

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

	return result, nil
}

/**
* executeDelete
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) executeDelete(tx *Tx) ([]et.Json, error) {
	model := s.model
	if model == nil {
		return nil, fmt.Errorf(msg.MSG_MODEL_IS_NIL)
	}

	s.wheres.SetOwner(model)
	items, err := s.wheres.Run(tx)
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

	return result, nil
}

/**
* executeUpsert
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) executeUpsert(tx *Tx) ([]et.Json, error) {
	model := s.model
	if model == nil {
		return nil, fmt.Errorf(msg.MSG_MODEL_IS_NIL)
	}

	new := s.data

	exists := true
	for _, name := range model.PrimaryKeys {
		source, ok := model.stores[name]
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
		s.wheres.Add(Eq(name, key))
	}

	if !exists {
		result, err := s.executeInsert(tx)
		if err != nil {
			return nil, err
		}
		return []et.Json{result}, nil
	}

	return s.executeUpdate(tx)
}

/**
* Execute: Executes the command
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Cmd) Execute(tx *Tx) ([]et.Json, error) {
	tx, commit := getTx(tx)
	tx.isDebug = s.isDebug
	result := []et.Json{}
	switch s.command {
	case INSERT:
		item, err := s.executeInsert(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	case UPDATE:
		items, err := s.executeUpdate(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, items...)
	case DELETE:
		items, err := s.executeDelete(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, items...)
	case UPSERT:
		items, err := s.executeUpsert(tx)
		if err != nil {
			return nil, err
		}

		result = append(result, items...)
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
* Insert: Inserts the model
* @param new et.Json
* @return *Cmd
**/
func (s *Cmd) Insert(new et.Json) *Cmd {
	s.command = INSERT
	s.data = new
	return s
}

/**
* Update: Updates the model
* @param data et.Json, where *Wheres
* @return *Cmd
**/
func (s *Cmd) Update(data et.Json) *Cmd {
	s.command = UPDATE
	s.data = data
	return s
}

/**
* Delete: Deletes the model
* @return *Cmd
**/
func (s *Cmd) Delete() *Cmd {
	s.command = DELETE
	return s
}

/**
* Upsert: Upserts the model
* @param new et.Json
* @return *Cmd
**/
func (s *Cmd) Upsert(new et.Json) *Cmd {
	s.command = UPSERT
	s.data = new
	return s
}

/**
* Where
* @param con *Condition
* @return *Cmd
**/
func (s *Cmd) Where(con *Condition) *Cmd {
	s.wheres.Add(con)
	return s
}

/**
* And
* @param con *Condition
* @return *Cmd
**/
func (s *Cmd) And(con *Condition) *Cmd {
	s.wheres.And(con)
	return s
}

/**
* Or
* @param con *Condition
* @return *Cmd
**/
func (s *Cmd) Or(con *Condition) *Cmd {
	s.wheres.Or(con)
	return s
}

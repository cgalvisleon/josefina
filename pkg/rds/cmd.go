package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/cgalvisleon/josefina/pkg/store"
)

type Command string

const (
	cmdInsert Command = "insert"
	cmdUpdate Command = "update"
	cmdDelete Command = "delete"
	cmdUpsert Command = "upsert"
)

type Cmd struct {
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
* putIndex
* @param store *store.FileStore, id string, key string
* @return string, error
**/
func (s *Cmd) putIndex(store *store.FileStore, id string, key any) (string, error) {
	result := map[string]bool{}
	_, err := store.Get(id, result)
	if err != nil {
		return id, err
	}

	st := fmt.Sprintf("%v", key)
	result[st] = true
	store.Put(id, result)

	return id, nil
}

/**
* deleteIndex
* @param store *store.FileStore, id string, key string
* @return bool, error
**/
func (s *Cmd) deleteIndex(store *store.FileStore, id string, key any) (bool, error) {
	result := map[string]bool{}
	exists, err := store.Get(id, result)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	st := fmt.Sprintf("%v", key)
	if _, ok := result[st]; !ok {
		return false, nil
	}

	delete(result, st)
	if len(result) == 0 {
		store.Delete(id)
		return true, nil
	}

	store.Put(id, result)
	return true, nil
}

/**
* insert: Inserts the model
* @param ctx *Tx, new et.Json
* @return et.Items, error
**/
func (s *Cmd) insert(ctx *Tx, new et.Json) (et.Items, error) {
	model := s.model
	idx, ok := new[INDEX]
	if !ok {
		idx = model.getJid()
		new[INDEX] = idx
	}

	// Validate required fields
	for _, name := range model.Required {
		if _, ok := new[name]; !ok {
			return et.Items{}, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
	}

	// Validate unique fields
	for _, name := range model.Unique {
		if _, ok := new[name]; !ok {
			return et.Items{}, fmt.Errorf(msg.MSG_FIELD_REQUIRED, name)
		}
		source := model.data[name]
		key := fmt.Sprintf("%v", new[name])
		if source.IsExist(key) {
			return et.Items{}, fmt.Errorf(msg.MSG_RECORD_EXISTS)
		}
	}

	// Run before insert triggers
	for _, trigger := range s.beforeInserts {
		err := s.runTrigger(trigger, ctx, et.Json{}, new)
		if err != nil {
			return et.Items{}, err
		}
	}

	// Insert data into indexes
	for _, name := range model.Indexes {
		source := model.data[name]
		key := fmt.Sprintf("%v", new[name])
		if key == "" {
			continue
		}
		if name == INDEX {
			source.Put(key, new)
		} else {
			s.putIndex(source, key, idx)
		}
	}

	// Run after insert triggers
	for _, trigger := range s.afterInserts {
		err := s.runTrigger(trigger, ctx, et.Json{}, new)
		if err != nil {
			return et.Items{}, err
		}
	}

	result := et.Items{}
	result.Add(new)
	return result, nil
}

/**
* update: Updates the model
* @param ctx *Tx, data et.Json, where *Wheres
* @return et.Items, error
**/
func (s *Cmd) update(ctx *Tx, data et.Json, where *Wheres) (et.Items, error) {
	result := et.Items{}
	model := s.model
	items, err := where.Rows()
	if err != nil {
		return result, err
	}

	for _, old := range items {
		// Get index
		idx, ok := old[INDEX]
		if !ok {
			return result, errorRecordNotFound
		}

		// Update data
		new := old.Clone()
		for k, v := range data {
			new[k] = v
		}

		// Run before update triggers
		for _, trigger := range s.beforeUpdates {
			err := s.runTrigger(trigger, ctx, old, new)
			if err != nil {
				return et.Items{}, err
			}
		}

		// Insert data into indexes
		for _, name := range model.Indexes {
			source := model.data[name]
			key := fmt.Sprintf("%v", new[name])
			if key == "" {
				continue
			}
			if name == INDEX {
				source.Put(key, new)
			} else {
				s.putIndex(source, key, idx)
			}
		}

		// Run after update triggers
		for _, trigger := range s.afterUpdates {
			err := s.runTrigger(trigger, ctx, old, new)
			if err != nil {
				return et.Items{}, err
			}
		}

		result.Add(new)
	}

	return result, nil
}

/**
* delete: Deletes the model
* @param ctx *Tx, where *Wheres
* @return et.Items, error
**/
func (s *Cmd) delete(ctx *Tx, where *Wheres) (et.Items, error) {
	result := et.Items{}
	model := s.model
	new := et.Json{}

	items, err := where.Rows()
	if err != nil {
		return result, err
	}

	for _, old := range items {
		// Get index
		idx, ok := old[INDEX]
		if !ok {
			return result, errorRecordNotFound
		}

		// Run before delete triggers
		for _, trigger := range s.beforeDeletes {
			err := s.runTrigger(trigger, ctx, old, new)
			if err != nil {
				return et.Items{}, err
			}
		}

		// Delete data from indexes
		for _, name := range model.Indexes {
			source := model.data[name]
			key := fmt.Sprintf("%v", new[name])
			if key == "" {
				continue
			}
			if name == INDEX {
				source.Delete(key)
			} else {
				s.deleteIndex(source, key, idx)
			}
		}

		// Run after delete triggers
		for _, trigger := range s.afterDeletes {
			err := s.runTrigger(trigger, ctx, old, new)
			if err != nil {
				return et.Items{}, err
			}
		}

		result.Add(old)
	}

	return result, nil
}

/**
* upsert: Upserts the model
* @param ctx *Tx, new et.Json
* @return et.Items, error
**/
func (s *Cmd) upsert(ctx *Tx, new et.Json) (et.Items, error) {
	model := s.model
	where := newWhere(model)
	exists := true
	for _, name := range model.PrimaryKeys {
		source, ok := model.data[name]
		if !ok {
			return et.Items{}, errorPrimaryKeysNotFound
		}
		key := fmt.Sprintf("%v", new[name])
		if key == "" {
			return et.Items{}, errorPrimaryKeysNotFound
		}
		where.Add(Eq(name, key))
		if !source.IsExist(key) {
			exists = false
			break
		}
	}

	if !exists {
		return s.insert(ctx, new)
	}

	return s.update(ctx, new, where)
}

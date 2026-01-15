package josefina

import (
	"slices"

	"github.com/cgalvisleon/et/et"
)

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type TriggerFunction func(tx *Tx, old, new et.Json) error

/**
* runTrigger
* @param trigger *Trigger, tx *Tx, old et.Json, new et.Json
* @return error
**/
func (s *Model) runTrigger(trigger *Trigger, tx *Tx, old, new et.Json) error {
	vm, ok := s.triggers[trigger.Name]
	if !ok {
		vm = newVm()
		s.triggers[trigger.Name] = vm
	}

	vm.Set("self", s)
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
* addBeforeInsert
* @param name string, fn []byte
* @return void
**/
func (s *Model) addBeforeInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeInserts[idx].Definition = fn
	}
	s.BeforeInserts = append(s.BeforeInserts, &Trigger{Name: name, Definition: fn})
}

/**
* addAfterInsert
* @param name string, fn []byte
* @return void
**/
func (s *Model) addAfterInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterInserts[idx].Definition = fn
	}
	s.AfterInserts = append(s.AfterInserts, &Trigger{Name: name, Definition: fn})
}

/**
* addBeforeUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) addBeforeUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeUpdates[idx].Definition = fn
	}
	s.BeforeUpdates = append(s.BeforeUpdates, &Trigger{Name: name, Definition: fn})
}

/**
* addAfterUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) addAfterUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterUpdates[idx].Definition = fn
	}
	s.AfterUpdates = append(s.AfterUpdates, &Trigger{Name: name, Definition: fn})
}

/**
* addBeforeDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) addBeforeDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeDeletes[idx].Definition = fn
	}
	s.BeforeDeletes = append(s.BeforeDeletes, &Trigger{Name: name, Definition: fn})
}

/**
* addAfterDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) addAfterDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterDeletes[idx].Definition = fn
	}
	s.AfterDeletes = append(s.AfterDeletes, &Trigger{Name: name, Definition: fn})
}

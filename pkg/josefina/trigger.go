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
* defaultBeforeInsert
* @param tx *Tx, old et.Json, new et.Json
* @return error
**/
func defaultBeforeInsert(tx *Tx, old, new et.Json) error {
	return nil
}

/**
* defaultAfterInsert
* @param tx *Tx, old et.Json, new et.Json
* @return error
**/
func defaultAfterInsert(tx *Tx, old, new et.Json) error {
	return nil
}

/**
* defaultBeforeUpdate
* @param tx *Tx, old et.Json, new et.Json
* @return error
**/
func defaultBeforeUpdate(tx *Tx, old, new et.Json) error {
	return nil
}

/**
* defaultAfterUpdate
* @param tx *Tx, old et.Json, new et.Json
* @return error
**/
func defaultAfterUpdate(tx *Tx, old, new et.Json) error {
	return nil
}

/**
* defaultBeforeDelete
* @param tx *Tx, old et.Json, new et.Json
* @return error
**/
func defaultBeforeDelete(tx *Tx, old, new et.Json) error {
	return nil
}

/**
* defaultAfterDelete
* @param tx *Tx, old et.Json, new et.Json
* @return error
**/
func defaultAfterDelete(tx *Tx, old, new et.Json) error {
	return nil
}

/**
* addBeforeInsert
* @param name string, fn TriggerFunction
* @return void
**/
func (s *Model) addBeforeInsert(name string, fn TriggerFunction) {
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		return
	}
	s.beforeInserts = append(s.beforeInserts, fn)
}

/**
* addAfterInsert
* @param name string, fn TriggerFunction
* @return void
**/
func (s *Model) addAfterInsert(name string, fn TriggerFunction) {
	idx := slices.IndexFunc(s.AfterInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		return
	}
	s.afterInserts = append(s.afterInserts, fn)
}

/**
* addBeforeUpdate
* @param name string, fn TriggerFunction
* @return void
**/
func (s *Model) addBeforeUpdate(name string, fn TriggerFunction) {
	idx := slices.IndexFunc(s.BeforeUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		return
	}
	s.beforeUpdates = append(s.beforeUpdates, fn)
}

/**
* addAfterUpdate
* @param name string, fn TriggerFunction
* @return void
**/
func (s *Model) addAfterUpdate(name string, fn TriggerFunction) {
	idx := slices.IndexFunc(s.AfterUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		return
	}
	s.afterUpdates = append(s.afterUpdates, fn)
}

/**
* addBeforeDelete
* @param name string, fn TriggerFunction
* @return void
**/
func (s *Model) addBeforeDelete(name string, fn TriggerFunction) {
	idx := slices.IndexFunc(s.BeforeDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		return
	}
	s.beforeDeletes = append(s.beforeDeletes, fn)
}

/**
* addAfterDelete
* @param name string, fn TriggerFunction
* @return void
**/
func (s *Model) addAfterDelete(name string, fn TriggerFunction) {
	idx := slices.IndexFunc(s.AfterDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		return
	}
	s.afterDeletes = append(s.afterDeletes, fn)
}

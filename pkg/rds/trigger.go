package rds

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

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
* AddBeforeInsert
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeInserts[idx].Definition = fn
	}
	s.BeforeInserts = append(s.BeforeInserts, &Trigger{Name: name, Definition: fn})
}

/**
* AddAfterInsert
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterInsert(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterInserts, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterInserts[idx].Definition = fn
	}
	s.AfterInserts = append(s.AfterInserts, &Trigger{Name: name, Definition: fn})
}

/**
* AddBeforeUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeUpdates[idx].Definition = fn
	}
	s.BeforeUpdates = append(s.BeforeUpdates, &Trigger{Name: name, Definition: fn})
}

/**
* AddAfterUpdate
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterUpdate(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterUpdates, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterUpdates[idx].Definition = fn
	}
	s.AfterUpdates = append(s.AfterUpdates, &Trigger{Name: name, Definition: fn})
}

/**
* AddBeforeDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddBeforeDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.BeforeDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.BeforeDeletes[idx].Definition = fn
	}
	s.BeforeDeletes = append(s.BeforeDeletes, &Trigger{Name: name, Definition: fn})
}

/**
* AddAfterDelete
* @param name string, fn []byte
* @return void
**/
func (s *Model) AddAfterDelete(name string, fn []byte) {
	idx := slices.IndexFunc(s.AfterDeletes, func(t *Trigger) bool { return t.Name == name })
	if idx != -1 {
		s.AfterDeletes[idx].Definition = fn
	}
	s.AfterDeletes = append(s.AfterDeletes, &Trigger{Name: name, Definition: fn})
}

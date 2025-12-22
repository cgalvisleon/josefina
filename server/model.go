package server

import (
	"github.com/cgalvisleon/et/et"
)

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type TriggerFunction func(tx *Tx, old, new et.Json) error

type Model struct {
	Database      string             `json:"database"`
	Schema        string             `json:"schema"`
	Name          string             `json:"name"`
	Files         []string           `json:"fields"`
	Columns       []*Column          `json:"columns"`
	PrimaryKeys   []string           `json:"primary_keys"`
	Unique        []string           `json:"unique"`
	Indexes       []string           `json:"indexes"`
	Required      []string           `json:"required"`
	Hidden        []string           `json:"hidden"`
	Master        map[string]*Master `json:"master"`
	Details       map[string]*Detail `json:"details"`
	Rollups       map[string]*Detail `json:"rollups"`
	Relations     map[string]*Detail `json:"relations"`
	BeforeInserts []*Trigger         `json:"before_inserts"`
	BeforeUpdates []*Trigger         `json:"before_updates"`
	BeforeDeletes []*Trigger         `json:"before_deletes"`
	AfterInserts  []*Trigger         `json:"after_inserts"`
	AfterUpdates  []*Trigger         `json:"after_updates"`
	AfterDeletes  []*Trigger         `json:"after_deletes"`
	IsLocked      bool               `json:"is_locked"`
	Version       int                `json:"version"`
	IsCore        bool               `json:"is_core"`
	IsDebug       bool               `json:"-"`
	isInit        bool               `json:"-"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Model) ToJson() et.Json {
	return ToJson(s)
}

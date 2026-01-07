package model

import (
	"github.com/cgalvisleon/josefina/server/store"
)

type Model struct {
	Database      string                      `json:"database"`
	Schema        string                      `json:"schema"`
	Name          string                      `json:"name"`
	Data          *store.FileStore            `json:"data"`
	Columns       []*Column                   `json:"columns"`
	Indexes       map[string]*store.FileStore `json:"indexes"`
	PrimaryKeys   []string                    `json:"primary_keys"`
	Unique        []string                    `json:"unique"`
	Required      []string                    `json:"required"`
	Hidden        []string                    `json:"hidden"`
	References    []string                    `json:"references"`
	Master        map[string]*Master          `json:"master"`
	Details       map[string]*Detail          `json:"details"`
	Rollups       map[string]*Detail          `json:"rollups"`
	Relations     map[string]*Detail          `json:"relations"`
	BeforeInserts []*Trigger                  `json:"before_inserts"`
	BeforeUpdates []*Trigger                  `json:"before_updates"`
	BeforeDeletes []*Trigger                  `json:"before_deletes"`
	AfterInserts  []*Trigger                  `json:"after_inserts"`
	AfterUpdates  []*Trigger                  `json:"after_updates"`
	AfterDeletes  []*Trigger                  `json:"after_deletes"`
	IsLocked      bool                        `json:"is_locked"`
	Version       int                         `json:"version"`
	IsCore        bool                        `json:"is_core"`
	IsDebug       bool                        `json:"-"`
	isInit        bool                        `json:"-"`
}

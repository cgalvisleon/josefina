package model

import (
	"github.com/cgalvisleon/josefina/server/store"
)

type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
}

type Model struct {
	From          *From                       `json:"from"`
	Data          *store.FileStore            `json:"data"`
	Fields        []*Field                    `json:"fields"`
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
	IsStrict      bool                        `json:"is_strict"`
	Version       int                         `json:"version"`
	Host          string                      `json:"-"`
	IsCore        bool                        `json:"is_core"`
	IsDebug       bool                        `json:"-"`
	isInit        bool                        `json:"-"`
}

func (s *Model) Init() error {
	if s.isInit {
		return nil
	}

	s.isInit = true
	return nil
}

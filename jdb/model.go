package jdb

import (
	"encoding/json"
	"regexp"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type TriggerFunction func(tx *Tx, old, new et.Json) error
type DataContext func(tx *Tx, data et.Json)

type Model struct {
	DB            *DB                    `json:"-"`
	Schema        string                 `json:"schema"`
	Name          string                 `json:"name"`
	Table         string                 `json:"table"`
	Columns       []*Column              `json:"columns"`
	SourceField   *Column                `json:"source_field"`
	IndexField    *Column                `json:"index_field"`
	PrimaryKeys   []string               `json:"primary_keys"`
	Unique        []string               `json:"unique"`
	Indexes       []string               `json:"indexes"`
	Required      []string               `json:"required"`
	Hidden        []string               `json:"hidden"`
	Master        map[string]*Detail     `json:"master"`
	Details       map[string]*Detail     `json:"details"`
	Rollups       map[string]*Detail     `json:"rollups"`
	Relations     map[string]*Detail     `json:"relations"`
	BeforeInserts []*Trigger             `json:"before_inserts"`
	BeforeUpdates []*Trigger             `json:"before_updates"`
	BeforeDeletes []*Trigger             `json:"before_deletes"`
	AfterInserts  []*Trigger             `json:"after_inserts"`
	AfterUpdates  []*Trigger             `json:"after_updates"`
	AfterDeletes  []*Trigger             `json:"after_deletes"`
	IsLocked      bool                   `json:"is_locked"`
	Version       int                    `json:"version"`
	IsCore        bool                   `json:"is_core"`
	IsDebug       bool                   `json:"-"`
	isInit        bool                   `json:"-"`
	calcs         map[string]DataContext `json:"-"`
	beforeInserts []TriggerFunction      `json:"-"`
	beforeUpdates []TriggerFunction      `json:"-"`
	beforeDeletes []TriggerFunction      `json:"-"`
	afterInserts  []TriggerFunction      `json:"-"`
	afterUpdates  []TriggerFunction      `json:"-"`
	afterDeletes  []TriggerFunction      `json:"-"`
}

/**
* Serialize
* @return []byte, error
**/
func (s *Model) Serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Model) ToJson() et.Json {
	bt, err := s.Serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* Save
* @return error
**/
func (s *Model) Save() error {
	if models == nil {
		return nil
	}

	serialize, err := s.Serialize()
	if err != nil {
		return err
	}

	now := timezone.Now()
	_, err = models.
		Upsert(et.Json{
			"name":       s.Name,
			"version":    s.Version,
			"definition": serialize,
		}).
		BeforeInsertOrUpdate(func(tx *Tx, old, new et.Json) error {
			new.Set("created_at", now)
			new.Set("updated_at", now)
			return nil
		}).
		BeforeUpdate(func(tx *Tx, old, new et.Json) error {
			new.Set("updated_at", now)
			return nil
		}).
		Exec()
	if err != nil {
		return err
	}

	return nil
}

/**
* Init
* @return error
**/
func (s *Model) Init() error {
	if s.isInit {
		return nil
	}

	s.isInit = true
	return nil
}

/**
* idxColumn
* @param name string
* @return int
**/
func (s *Model) idxColumn(name string) int {
	return slices.IndexFunc(s.Columns, func(column *Column) bool { return column.Name == name })
}

/**
* FindField
* @param name string
* @return *Field
**/
func (s *Model) FindColumn(name string) *Column {
	idx := s.idxColumn(name)
	if idx != -1 {
		return s.Columns[idx]
	}

	if s.IsLocked {
		return nil
	}

	if s.SourceField == nil {
		return nil
	}

	return newColumn(s, name, TpAtrib, TpAny, "", []byte{})
}

/**
* FindField
* @param name string
* @return *Field
**/
func (s *Model) FindField(name string) *Field {
	pattern1 := regexp.MustCompile(`^([A-Za-z0-9>]+):([A-Za-z0-9]+)$`) // name:as
	pattern2 := regexp.MustCompile(`^([A-Za-z0-9>]+)$`)                // name

	if pattern1.MatchString(name) {
		matches := pattern1.FindStringSubmatch(name)
		if len(matches) == 3 {
			name = matches[1]
			as := matches[2]
			column := s.FindColumn(name)
			if column != nil {
				result := column.Field()
				result.As = as
				return result
			}
		}
	} else if pattern2.MatchString(name) {
		as := name
		column := s.FindColumn(name)
		if column != nil {
			result := column.Field()
			result.As = as
			return result
		}
	}

	return nil
}

/**
* GetId
* @param id string
* @return string
**/
func (s *Model) GetId(id string) string {
	return reg.TagULID(s.Name, id)
}

/**
* Insert
* @param data et.Json
* @return *Cmd
**/
func (s *Model) Insert(data et.Json) *Cmd {
	result := newCommand(s, TypeInsert)
	result.Data = append(result.Data, data)
	return result
}

/**
* Update
* @param data et.Json
* @return *Cmd
**/
func (s *Model) Update(data et.Json) *Cmd {
	result := newCommand(s, TypeUpdate)
	result.Data = append(result.Data, data)
	return result
}

/**
* Delete
* @return *Cmd
**/
func (s *Model) Delete() *Cmd {
	result := newCommand(s, TypeDelete)
	return result
}

/**
* Upsert
* @param data et.Json
* @return *Cmd
**/
func (s *Model) Upsert(data et.Json) *Cmd {
	result := newCommand(s, TypeUpsert)
	result.Data = append(result.Data, data)
	return result
}

/**
* Select
* @param fields ...string
* @return *Ql
**/
func (s *Model) Select(fields ...string) *Ql {
	result := From(s, "A")
	for _, field := range fields {
		fld := s.FindField(field)
		if fld != nil {
			result.Selects = append(result.Selects, fld)
		}
	}

	return result
}

/**
* SelectColumns
* @return []string
**/
func (s *Model) SelectColumns() []string {
	result := []string{}
	for _, col := range s.Columns {
		if col.TypeColumn == TpColumn {
			result = append(result, col.Name)
		}
	}
	return result
}

/**
* Counted
* @return (int, error)
**/
func (s *Model) Counted() (int, error) {
	result := From(s, "A")
	return result.Count()
}

/**
* Where
* @param condition *Condition
* @return *Ql
**/
func (s *Model) Where(condition *Condition) *Ql {
	result := From(s, "A")
	result.Wheres.Add(condition)
	return result
}

/**
* WhereByPrimaryKeys
* @param data et.Json
* @return *Ql
**/
func (s *Model) WhereByPrimaryKeys(data et.Json) *Ql {
	result := From(s, "A")
	for _, col := range s.PrimaryKeys {
		val := data[col]
		if val == nil {
			continue
		}
		result.Where(Eq(col, val))
	}
	return result
}

/**
* WhereByCommand
* @param cmd *Cmd
* @return *Ql
**/
func (s *Model) WhereByCommand(cmd *Cmd) *Ql {
	result := From(s, "A")
	for _, cond := range cmd.Wheres.Conditions {
		result.Where(cond)
	}
	return result
}

/**
* Current
* @return *Ql
**/
func (s *Model) Current(where *Wheres) *Ql {
	result := From(s, "A")
	fields := s.SelectColumns()
	result.Select(fields...)
	for _, cond := range where.Conditions {
		result.Where(cond)
	}
	return result
}

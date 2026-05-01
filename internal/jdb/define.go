package jdb

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/internal/msg"
)

/**
* defineField: Defines the field
* @param name string, tpField TypeField, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func (s *Model) defineField(name string, tpField TypeField, tpData TypeData, defaultValue interface{}) (*Field, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(string(tpField), 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "tpField")
	}
	if !utility.ValidStr(string(tpData), 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "type")
	}

	result, ok := s.Fields[name]
	if ok {
		result.TypeField = tpField
		result.TypeData = tpData
		result.DefaultValue = defaultValue
		return result, nil
	}

	result, err := newField(s, name, tpField, tpData, defaultValue)
	if err != nil {
		return nil, err
	}
	s.Fields[name] = result

	return result, nil
}

/**
* findIndex: Finds the index
* @param name string
* @return *Index, bool
**/
func (s *Model) findIndex(name string) (*Index, bool) {
	idx := slices.IndexFunc(s.Indexes, func(i *Index) bool { return strings.EqualFold(i.Name, name) })
	if idx == -1 {
		return nil, false
	}
	return s.Indexes[idx], true
}

/**
* defineSource: Defines the source field
* @return *Field, error
**/
func (s *Model) defineSource() (*Field, error) {
	result, err := s.defineField(INDEX, TpAtrib, TpJson, "")
	if err != nil {
		return nil, err
	}
	s.DefineIndex(INDEX, TpIndexHash)
	s.DefineHidden(INDEX)
	return result, nil
}

/**
* DefineField: Defines the field
* @param name string, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func (s *Model) DefineField(name string, tpData TypeData, defaultValue interface{}) (*Field, error) {
	result, err := s.defineField(name, TpAtrib, tpData, defaultValue)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
* DefineIndex: Defines the index
* @param name string, tp TpIndex
* @return *Index, error
**/
func (s *Model) DefineIndex(name string, tp TpIndex) (*Index, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(string(tp), 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "type")
	}

	_, exists := s.Fields[name]
	if !exists {
		return nil, fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, name)
	}

	if tp == "" {
		tp = TpIndexHash
	}

	idx := slices.IndexFunc(s.Indexes, func(i *Index) bool { return strings.EqualFold(i.Name, name) })
	if idx == -1 {
		s.Indexes = append(s.Indexes, newIndex(s, name, tp))
		return s.Indexes[len(s.Indexes)-1], nil
	}
	return s.Indexes[idx], nil
}

/**
* DefineIndexes: Defines the index
* @param name string
* @return error
**/
func (s *Model) DefineIndexes(fields ...string) error {
	for _, field := range fields {
		_, err := s.DefineIndex(field, TpIndexHash)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
* DefineUnique: Defines the unique
* @param name string
* @return bool
**/
func (s *Model) DefineUnique(name string) (*Index, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	_, ok := s.Fields[name]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, name)
	}

	index, err := s.DefineIndex(name, TpIndexBTree)
	if err != nil {
		return nil, err
	}

	idx := slices.IndexFunc(s.Unique, func(i *Index) bool { return strings.EqualFold(i.Name, name) })
	if idx == -1 {
		s.Unique = append(s.Unique, index)
	}

	return index, nil
}

/**
* DefineUniques: Defines the unique
* @param fields ...string
* @return error
**/
func (s *Model) DefineUniques(fields ...string) error {
	for _, field := range fields {
		_, err := s.DefineUnique(field)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
* DefineRequired: Defines the required
* @param name string
* @return *Index, error
**/
func (s *Model) DefineRequired(name string) (*Index, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	_, ok := s.Fields[name]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, name)
	}

	index, err := s.DefineIndex(name, TpIndexBTree)
	if err != nil {
		return nil, err
	}

	idx := slices.IndexFunc(s.Required, func(i *Index) bool { return strings.EqualFold(i.Name, name) })
	if idx == -1 {
		s.Required = append(s.Required, index)
	}

	return index, nil
}

/**
* DefineRequired: Defines the required
* @param name string
* @return bool
**/
func (s *Model) DefineRequiredes(fields ...string) error {
	for _, field := range fields {
		_, err := s.DefineRequired(field)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
* DefineHidden: Defines the hidden
* @param name string
* @return bool
**/
func (s *Model) DefineHidden(name string) error {
	if !utility.ValidStr(name, 0, []string{""}) {
		return fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	_, ok := s.Fields[name]
	if !ok {
		return fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, name)
	}

	idx := slices.Index(s.Hidden, name)
	if idx == -1 {
		s.Hidden = append(s.Hidden, name)
	}

	return nil
}

/**
* DefineHiddens: Defines the hidden
* @param name string
* @return bool
**/
func (s *Model) DefineHiddens(fields ...string) error {
	for _, name := range fields {
		err := s.DefineHidden(name)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
* definePrimaryKey: Defines the primary keys
* @param name string
**/
func (s *Model) DefinePrimaryKeys(fields ...string) error {
	for _, field := range fields {
		_, ok := s.Fields[field]
		if !ok {
			return fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, field)
		}

		idx := slices.Index(s.PrimaryKeys, field)
		if idx == -1 {
			s.PrimaryKeys = append(s.PrimaryKeys, field)
			s.DefineRequired(field)
			s.DefineUnique(field)
		}
	}
	return nil
}

/**
* DefineForeignKeys: Defines the foreign keys
* @param name, key string, to *Model, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func (s *Model) DefineForeignKeys(to *Model, keys map[string]string, onDeleteCascade, onUpdateCascade bool) (*Detail, error) {
	for fk, pk := range keys {
		_, ok := s.Fields[pk]
		if !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, pk)
		}

		s.DefineRequired(pk)
		_, ok = to.Fields[fk]
		if !ok {
			return nil, fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, fk)
		}
		to.DefineRequired(fk)
	}

	name := fmt.Sprintf("%s_%s_fk", s.Name, to.Name)
	result := newDetail(to, keys, []string{}, onDeleteCascade, onUpdateCascade)
	s.ForeignKeys[name] = result
	return result, nil
}

/**
* DefineDetail: Defines the detail
* @param name string, keys map[string]string, version int
* @return *Model, error
**/
func (s *Model) DefineDetail(name string, keys map[string]string, version int) (*Model, error) {
	_, err := s.defineField(name, TpDetail, TpReference, []et.Json{})
	if err != nil {
		return nil, err
	}

	to, err := s.schema.NewModel(fmt.Sprintf("%s_%s", s.Name, name), false, version)
	if err != nil {
		return nil, err
	}

	forKeys := make(map[string]string)
	for fk, pk := range keys {
		_, err = s.DefineField(pk, TpKey, "")
		if err != nil {
			return nil, err
		}

		_, err = to.DefineField(fk, TpKey, "")
		if err != nil {
			return nil, err
		}

		s.DefinePrimaryKeys(pk)
		forKeys[pk] = fk
	}

	to.DefineForeignKeys(s, forKeys, true, true)
	s.Details[name] = newDetail(to, keys, []string{}, false, false)
	return to, nil
}

/**
* DefineRollup: Defines the rollup
* @param name string, to *Model, keys map[string]string, selects []string
* @return error
**/
func (s *Model) DefineRollup(name string, to *Model, keys map[string]string, selects []string) error {
	_, err := s.defineField(name, TpRollup, TpJson, []et.Json{})
	if err != nil {
		return err
	}

	s.Rollups[name] = newDetail(to, keys, selects, false, false)
	return nil
}

/**
* DefineRelation: Defines the relation
* @param to *Model, keys map[string]string, onDeleteCascade, onUpdateCascade bool
* @return error
**/
func (s *Model) DefineRelation(to *Model, keys map[string]string, onDeleteCascade, onUpdateCascade bool) error {
	detail := newDetail(to, keys, []string{}, onDeleteCascade, onUpdateCascade)
	s.Relations[to.Name] = detail
	return nil
}

/**
* DefineCalc: Defines the calc
* @param name string, definition []byte
* @return error
**/
func (s *Model) DefineCalc(name string, definition []byte) error {
	_, err := s.defineField(name, TpCalc, TpBytes, nil)
	if err != nil {
		return err
	}

	s.Calcs[name] = definition
	return nil
}

type DefineField struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"`
	Default interface{} `json:"default"`
}

type DefineIndex struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type DefineTo struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type DefineForeignKeys struct {
	To              *DefineTo         `json:"to"`
	Keys            map[string]string `json:"keys"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
}

type DefineDetail struct {
	Name    string            `json:"name"`
	Keys    map[string]string `json:"keys"`
	Version int               `json:"version"`
}

type DefineRollup struct {
	Name    string            `json:"name"`
	To      *DefineTo         `json:"to"`
	Keys    map[string]string `json:"keys"`
	Selects []string          `json:"selects"`
}

type DefineRelation struct {
	To              *DefineTo         `json:"to"`
	Keys            map[string]string `json:"keys"`
	OnDeleteCascade bool              `json:"on_delete_cascade"`
	OnUpdateCascade bool              `json:"on_update_cascade"`
}

type Define struct {
	Schema        string                  `json:"schema"`
	Name          string                  `json:"name"`
	Version       int                     `json:"version"`
	IsCore        bool                    `json:"is_core"`
	IsStrict      bool                    `json:"is_strict"`
	Fields        map[string]DefineField  `json:"fields"`
	Indexes       []DefineIndex           `json:"indexes"`
	PrimaryKeys   []string                `json:"primary_keys"`
	ForeignKeys   []DefineForeignKeys     `json:"foreign_keys"`
	Unique        []DefineIndex           `json:"unique"`
	Required      []DefineIndex           `json:"required"`
	Hidden        []string                `json:"hidden"`
	Details       map[string]DefineDetail `json:"details"`
	Rollups       map[string]DefineRollup `json:"rollups"`
	Relations     []DefineRelation        `json:"relations"`
	Calcs         map[string][]byte       `json:"calcs"`
	BeforeInserts []Trigger               `json:"before_inserts"`
	AfterInserts  []Trigger               `json:"after_inserts"`
	BeforeUpdates []Trigger               `json:"before_updates"`
	AfterUpdates  []Trigger               `json:"after_updates"`
	BeforeDeletes []Trigger               `json:"before_deletes"`
	AfterDeletes  []Trigger               `json:"after_deletes"`
}

/**
* ToJson
* @return et.Json, error
**/
func (s *Define) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}, err
	}

	result := et.Json{}
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

package jdb

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

/**
* defineColumn
* @param name string, tpColumn TypeColumn, tpData TypeData, hidden bool, defaultValue interface{}, definition []byte
* @return *Column
**/
func (s *Model) defineColumn(name string, tpColumn TypeColumn, tpData TypeData, hidden bool, defaultValue interface{}, definition []byte) (*Column, error) {
	if !utility.ValidStr(name, 0, []string{}) {
		return nil, fmt.Errorf(MSG_NAME_REQUIRED)
	}

	if !utility.ValidStr(tpColumn.Str(), 0, []string{}) {
		return nil, fmt.Errorf(MSG_TYPE_COLUMN_REQUIRED)
	}

	if !utility.ValidStr(tpData.Str(), 0, []string{}) {
		return nil, fmt.Errorf(MSG_TYPE_DATA_REQUIRED)
	}

	idx := s.idxColumn(name)
	if idx != -1 {
		return s.Columns[idx], nil
	}

	result := newColumn(s, name, tpColumn, tpData, defaultValue, definition)
	s.Columns = append(s.Columns, result)
	if hidden {
		s.Hidden = append(s.Hidden, name)
	}
	return result, nil
}

/**
* DefineIndex
* @param names ...string
* @return error
**/
func (s *Model) DefineIndex(names ...string) error {
	for _, name := range names {
		idx := slices.Index(s.Indexes, name)
		if idx != -1 {
			continue
		}

		idx = s.idxColumn(name)
		if idx == -1 {
			continue
		}

		s.Indexes = append(s.Indexes, name)
	}

	return nil
}

/**
* DefineColumn
* @param name string, tpData TypeData
* @return *Column
**/
func (s *Model) DefineColumn(name string, tpData TypeData) (*Column, error) {
	return s.defineColumn(name, TpColumn, tpData, false, nil, nil)
}

/**
* DefineSourceField
* @param name string
* @return error
**/
func (s *Model) DefineSourceField(name string) error {
	col, err := s.defineColumn(name, TpColumn, TpJson, false, nil, nil)
	if err != nil {
		return err
	}

	s.SourceField = col
	s.DefineIndex(name)
	return nil
}

/**
* DefineIndexField
* @param name string
* @return error
**/
func (s *Model) DefineIndexField(name string) error {
	col, err := s.defineColumn(name, TpColumn, TpJson, false, nil, nil)
	if err != nil {
		return err
	}

	s.IndexField = col
	s.DefineIndex(name)
	return nil
}

/**
* DefineAttribute
* @param name string, tpData TypeData, defaultValue interface{}
* @return *Column, error
**/
func (s *Model) DefineAttribute(name string, tpData TypeData, defaultValue interface{}) (*Column, error) {
	if s.SourceField == nil {
		s.DefineSourceField(SOURCE)
	}
	return s.defineColumn(name, TpAtrib, tpData, false, defaultValue, nil)
}

/**
* DefineRequired
* @param names ...string
* @return
**/
func (s *Model) DefineRequired(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.Required, name)
		if idx != -1 {
			continue
		}

		idx = s.idxColumn(name)
		if idx == -1 {
			continue
		}

		s.Required = append(s.Required, name)
	}
}

/**
* DefineUnique
* @param names ...string
* @return
**/
func (s *Model) DefineUnique(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.Unique, name)
		if idx != -1 {
			continue
		}

		idx = s.idxColumn(name)
		if idx == -1 {
			continue
		}

		s.Unique = append(s.Unique, name)
	}
}

/**
* DefinePrimaryKeys
* @param names ...string
* @return
**/
func (s *Model) DefinePrimaryKeys(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.PrimaryKeys, name)
		if idx != -1 {
			continue
		}

		idx = s.idxColumn(name)
		if idx == -1 {
			continue
		}

		s.DefineRequired(name)
		s.DefineUnique(name)
		s.PrimaryKeys = append(s.PrimaryKeys, name)
	}
}

/**
* DefineHidden
* @param names ...string
* @return
**/
func (s *Model) DefineHidden(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.Hidden, name)
		if idx != -1 {
			continue
		}

		idx = s.idxColumn(name)
		if idx == -1 {
			continue
		}
		s.Hidden = append(s.Hidden, name)
	}
}

/**
* DefineDetail
* @param name string, to *Model, keys map[string]string
* @return *Column
**/
func (s *Model) DefineDetail(name string, keys map[string]string) (*Model, error) {
	_, err := s.defineColumn(name, TpDetail, TpJson, false, []et.Json{}, nil)
	if err != nil {
		return nil, err
	}

	to, err := s.DB.NewModel(s.Schema, fmt.Sprintf("%s_%s", s.Name, name), 1)
	if err != nil {
		return nil, err
	}

	for pk, fk := range keys {
		_, err = s.defineColumn(pk, TpColumn, TpKey, false, "", nil)
		if err != nil {
			return nil, err
		}

		_, err = to.defineColumn(fk, TpColumn, TpKey, false, "", nil)
		if err != nil {
			return nil, err
		}
	}

	detail := newDetail(to, "", keys, []string{}, true, true)
	s.Details[name] = detail
	fkeys := map[string]string{}
	for k, v := range keys {
		fkeys[v] = k
	}
	to.Master[s.Name] = newDetail(s, "", fkeys, []string{}, true, true)
	return to, nil
}

/**
* DefineRollup
* @param name string, from string, keys map[string]string, selects []string
* @return *Model
**/
func (s *Model) DefineRollup(name, from string, keys map[string]string, selects []string) error {
	_, err := s.defineColumn(name, TpRollup, TpJson, false, []et.Json{}, nil)
	if err != nil {
		return err
	}

	to, err := s.DB.GetModel(from)
	if err != nil {
		return err
	}

	detail := newDetail(to, "", keys, selects, false, false)
	s.Rollups[name] = detail
	return nil
}

/**
* DefineRelation
* @param name string, from string, keys map[string]string
* @return *Model
**/
func (s *Model) DefineRelation(from string, keys map[string]string) error {
	to, err := s.DB.GetModel(from)
	if err != nil {
		return err
	}

	detail := newDetail(to, "", keys, []string{}, false, false)
	s.Relations[to.Name] = detail
	return nil
}

/**
* DefineCreatedAtField
* @return *Model
**/
func (s *Model) DefineCreatedAtField() *Model {
	s.DefineColumn(CREATED_AT, TpDateTime)
	return s
}

/**
* DefineUpdatedAtField
* @return *Model
**/
func (s *Model) DefineUpdatedAtField() *Model {
	s.DefineColumn(UPDATED_AT, TpDateTime)
	return s
}

/**
* DefineStatusFieldDefault
* @return *Model
**/
func (s *Model) DefineStatusFieldDefault() *Model {
	s.DefineColumn(STATUS, TpKey)
	return s
}

/**
* DefinePrimaryKeyField
* @return *Model
**/
func (s *Model) DefinePrimaryKeyField() *Model {
	s.DefinePrimaryKeys(KEY)
	return s
}

/**
* DefineSourceFieldDefault
* @return *Model
**/
func (s *Model) DefineSourceFieldDefault() *Model {
	s.DefineSourceField(SOURCE)
	return s
}

func (s *Model) DefineIndexFieldDefault() *Model {
	s.DefineIndexField(INDEX)
	return s
}

/**
* DefineModel
* @return *Model
**/
func (s *Model) DefineModel() *Model {
	s.DefineCreatedAtField()
	s.DefineUpdatedAtField()
	s.DefineStatusFieldDefault()
	s.DefinePrimaryKeyField()
	s.DefineSourceFieldDefault()
	s.DefineIndexFieldDefault()
	return s
}

/**
* DefineTenantModel
* @return *Model
**/
func (s *Model) DefineProjectModel() *Model {
	s.DefineCreatedAtField()
	s.DefineUpdatedAtField()
	s.DefineStatusFieldDefault()
	s.DefinePrimaryKeyField()
	s.DefineColumn(PROJECT_ID, TpKey)
	s.DefineSourceFieldDefault()
	s.DefineIndexFieldDefault()
	s.DefineIndex(PROJECT_ID)
	return s
}

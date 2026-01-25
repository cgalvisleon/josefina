package rds

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

func (s *Model) setChanged() {
	if !s.IsInit {
		return
	}
	s.changed = true
}

/**
* defineFields: Defines the fields
* @param name string, tpField TypeField, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func (s *Model) defineFields(name string, tpField TypeField, tpData TypeData, defaultValue interface{}) (*Field, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}

	result, ok := s.Fields[name]
	if ok {
		result.TypeField = tpField
		result.TypeData = tpData
		result.DefaultValue = defaultValue
		return result, nil
	}

	result, err := newField(s.From, name, tpField, tpData, defaultValue)
	if err != nil {
		return nil, err
	}
	s.Fields[name] = result
	s.setChanged()

	return result, nil
}

/**
* defineIndexe: Defines the index
* @param name string
**/
func (s *Model) defineIndexe(name string) bool {
	_, ok := s.Fields[name]
	if !ok {
		return false
	}

	idx := slices.Index(s.Indexes, name)
	if idx == -1 {
		s.Indexes = append(s.Indexes, name)
		s.setChanged()
	}
	return true
}

/*
*
* defineUnique: Defines the unique
* @param name string
* @return bool
*
 */
func (s *Model) defineUnique(name string) bool {
	_, ok := s.Fields[name]
	if !ok {
		return false
	}

	idx := slices.Index(s.Unique, name)
	if idx == -1 {
		s.Unique = append(s.Unique, name)
		s.defineIndexe(name)
	}
	return true
}

/**
* defineRequired: Defines the required
* @param name string
* @return bool
**/
func (s *Model) defineRequired(name string) bool {
	_, ok := s.Fields[name]
	if !ok {
		return false
	}

	idx := slices.Index(s.Required, name)
	if idx == -1 {
		s.Required = append(s.Required, name)
		s.defineIndexe(name)
	}
	return true
}

/**
* defineHidden: Defines the hidden
* @param name string
* @return bool
**/
func (s *Model) defineHidden(name string) bool {
	_, ok := s.Fields[name]
	if !ok {
		return false
	}

	idx := slices.Index(s.Hidden, name)
	if idx == -1 {
		s.Hidden = append(s.Hidden, name)
		s.setChanged()
	}
	return true
}

/**
* definePrimaryKey: Defines the primary keys
* @param name string
**/
func (s *Model) definePrimaryKey(name string) bool {
	_, ok := s.Fields[name]
	if !ok {
		return false
	}

	idx := slices.Index(s.PrimaryKeys, name)
	if idx == -1 {
		s.PrimaryKeys = append(s.PrimaryKeys, name)
		s.defineRequired(name)
		s.defineUnique(name)
	}
	return true
}

/**
* defineIndexField: Defines the index field
* @return *Field, error
**/
func (s *Model) defineIndexField() (*Field, error) {
	result, err := s.defineFields(INDEX, TpAtrib, TpKey, "")
	if err != nil {
		return nil, err
	}
	s.defineIndexe(INDEX)
	s.defineHidden(INDEX)
	return result, nil
}

/**
* DefineAtrib: Defines the field
* @param name string, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func (s *Model) DefineAtrib(name string, tpData TypeData, defaultValue interface{}) (*Field, error) {
	return s.defineFields(name, TpAtrib, tpData, defaultValue)
}

/**
* DefineReferences: Defines the references
* @param name, key string, to *Model, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func (s *Model) DefineReferences(name, key string, to *Model, onDeleteCascade, onUpdateCascade bool) (*Detail, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "name")
	}
	if !utility.ValidStr(key, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "key")
	}

	_, ok := s.Fields[name]
	if !ok {
		return nil, fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, name)
	}

	result, ok := s.References[name]
	if ok {
		result.OnDeleteCascade = onDeleteCascade
		result.OnUpdateCascade = onUpdateCascade
		return result, nil
	}

	to.defineIndexe(name)
	to.defineRequired(name)
	result = newDetail(to, map[string]string{name: key}, []string{}, onDeleteCascade, onUpdateCascade)
	s.References[name] = result
	return result, nil
}

/**
* DefineDetail: Defines the detail
* @param name string, keys map[string]string, version int
* @return *Model, error
**/
func (s *Model) DefineDetail(name string, keys map[string]string, version int) (*Model, error) {
	_, err := s.defineFields(name, TpDetail, TpAny, []et.Json{})
	if err != nil {
		return nil, err
	}

	to, err := newModel(s.Database, s.Schema, fmt.Sprintf("%s_%s", s.Name, name), false, version)
	if err != nil {
		return nil, err
	}

	for fk, pk := range keys {
		_, err = s.DefineAtrib(pk, TpKey, "")
		if err != nil {
			return nil, err
		}

		_, err = to.DefineAtrib(fk, TpKey, "")
		if err != nil {
			return nil, err
		}

		s.definePrimaryKey(pk)
		_, err = to.DefineReferences(fk, pk, s, true, true)
		if err != nil {
			return nil, err
		}
	}

	s.Details[name] = newDetail(to, keys, []string{}, false, false)
	return to, nil
}

/**
* DefineRollup: Defines the rollup
* @param name string, from string, keys map[string]string, selects []string
* @return *Model
**/
func (s *Model) DefineRollup(name string, from *From, keys map[string]string, selects []string) (*Model, error) {
	_, err := s.defineFields(name, TpRollup, TpJson, []et.Json{})
	if err != nil {
		return nil, err
	}

	to, err := getModel(from)
	if err != nil {
		return nil, err
	}

	s.Rollups[name] = newDetail(to, keys, selects, false, false)
	return to, nil
}

/**
* DefineRelation: Defines the relation
* @param from *From, keys map[string]string, onDeleteCascade, onUpdateCascade bool
* @return error
**/
func (s *Model) DefineRelation(from *From, keys map[string]string, onDeleteCascade, onUpdateCascade bool) error {
	to, err := getModel(from)
	if err != nil {
		return err
	}

	detail := newDetail(to, keys, []string{}, onDeleteCascade, onUpdateCascade)
	s.Relations[to.Name] = detail
	s.setChanged()
	return nil
}

/**
* DefineCalc: Defines the calc
* @param name string, definition []byte
* @return error
**/
func (s *Model) DefineCalc(name string, definition []byte) error {
	_, err := s.defineFields(name, TpCalc, TpBytes, nil)
	if err != nil {
		return err
	}

	s.Calcs[name] = definition
	s.setChanged()
	return nil
}

/**
* DefineIndexes: Defines the indexes
* @param names ...string
* @return error
**/
func (s *Model) DefineIndexes(names ...string) bool {
	for _, name := range names {
		ok := s.defineIndexe(name)
		if !ok {
			return false
		}
	}
	return true
}

/**
* DefinePrimaryKeys: Defines the primary keys
* @param names ...string
* @return bool
**/
func (s *Model) DefinePrimaryKeys(names ...string) bool {
	for _, name := range names {
		ok := s.definePrimaryKey(name)
		if !ok {
			return false
		}
	}
	return true
}

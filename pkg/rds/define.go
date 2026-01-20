package rds

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* existsField: Checks if the field exists
* @param name string
* @return bool
**/
func (s *Model) existsField(name string) bool {
	_, ok := s.Fields[name]
	return ok
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
* defineReferences: Defines the references
* @param name, key string, to *Model, onDeleteCascade, onUpdateCascade bool
* @return *Detail
**/
func (s *Model) defineReferences(name, key string, to *Model, onDeleteCascade, onUpdateCascade bool) (*Detail, error) {
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
* defineAtrib: Defines the field
* @param name string, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func (s *Model) defineAtrib(name string, tpData TypeData, defaultValue interface{}) (*Field, error) {
	return s.defineFields(name, TpAtrib, tpData, defaultValue)
}

/**
* defineDetail: Defines the detail
* @param name string, keys map[string]string, version int
* @return *Model, error
**/
func (s *Model) defineDetail(name string, keys map[string]string, version int) (*Model, error) {
	_, err := s.defineFields(name, TpDetail, TpAny, []et.Json{})
	if err != nil {
		return nil, err
	}

	to, err := s.db.newModel(s.Schema, fmt.Sprintf("%s_%s", s.Name, name), false, version)
	if err != nil {
		return nil, err
	}

	for fk, pk := range keys {
		_, err = s.defineAtrib(pk, TpKey, "")
		if err != nil {
			return nil, err
		}

		_, err = to.defineAtrib(fk, TpKey, "")
		if err != nil {
			return nil, err
		}

		s.definePrimaryKey(pk)
		_, err = to.defineReferences(fk, pk, s, true, true)
		if err != nil {
			return nil, err
		}
	}

	s.Details[name] = newDetail(to, keys, []string{}, false, false)
	return to, nil
}

/**
* defineRollup: Defines the rollup
* @param name string, from string, keys map[string]string, selects []string
* @return *Model
**/
func (s *Model) defineRollup(name, from string, keys map[string]string, selects []string) (*Model, error) {
	_, err := s.defineFields(name, TpRollup, TpJson, []et.Json{})
	if err != nil {
		return nil, err
	}

	to, err := s.db.getModel(s.Schema, from)
	if err != nil {
		return nil, err
	}

	s.Rollups[name] = newDetail(to, keys, selects, false, false)
	return to, nil
}

/**
* defineRelation: Defines the relation
* @param from string, keys map[string]string, onDeleteCascade bool, onUpdateCascade bool
* @return *Model
**/
func (s *Model) defineRelation(from string, keys map[string]string, onDeleteCascade, onUpdateCascade bool) error {
	to, err := s.db.getModel(s.Schema, from)
	if err != nil {
		return err
	}

	detail := newDetail(to, keys, []string{}, onDeleteCascade, onUpdateCascade)
	s.Relations[to.Name] = detail
	return nil
}

/**
* defineCalc: Defines the calc
* @param name string, definition []byte
* @return error
**/
func (s *Model) defineCalc(name string, definition []byte) error {
	_, err := s.defineFields(name, TpCalc, TpBytes, nil)
	if err != nil {
		return err
	}

	s.Calcs[name] = definition
	return nil
}

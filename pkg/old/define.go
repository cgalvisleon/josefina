package old

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
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
* DefineIndexes: Defines the index
* @param name string
**/
func (s *Model) DefineIndexes(fields ...string) error {
	for _, field := range fields {
		_, ok := s.Fields[field]
		if !ok {
			return fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, field)
		}

		idx := slices.Index(s.Indexes, field)
		if idx == -1 {
			s.Indexes = append(s.Indexes, field)
		}
	}

	return nil
}

/*
*
* DefineUnique: Defines the unique
* @param name string
* @return bool
*
 */
func (s *Model) DefineUnique(fields ...string) error {
	for _, field := range fields {
		_, ok := s.Fields[field]
		if !ok {
			return fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, field)
		}

		idx := slices.Index(s.Unique, field)
		if idx == -1 {
			s.Unique = append(s.Unique, field)
			s.DefineIndexes(field)
		}
	}
	return nil
}

/**
* DefineRequired: Defines the required
* @param name string
* @return bool
**/
func (s *Model) DefineRequired(fields ...string) error {
	for _, field := range fields {
		_, ok := s.Fields[field]
		if !ok {
			return fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, field)
		}

		idx := slices.Index(s.Required, field)
		if idx == -1 {
			s.Required = append(s.Required, field)
			s.DefineIndexes(field)
		}
	}
	return nil
}

/**
* DefineHidden: Defines the hidden
* @param name string
* @return bool
**/
func (s *Model) DefineHidden(fields ...string) error {
	for _, field := range fields {
		_, ok := s.Fields[field]
		if !ok {
			return fmt.Errorf(msg.MSG_FIELD_NOT_FOUND, field)
		}

		idx := slices.Index(s.Hidden, field)
		if idx == -1 {
			s.Hidden = append(s.Hidden, field)
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
	result := newDetail(to.From, keys, []string{}, onDeleteCascade, onUpdateCascade)
	s.ForeignKeys[name] = result
	return result, nil
}

/**
* defineIndexField: Defines the index field
* @return *Field, error
**/
func (s *Model) defineIndexField() (*Field, error) {
	result, err := s.defineField(INDEX, TpAtrib, TpKey, "")
	if err != nil {
		return nil, err
	}
	s.DefineIndexes(INDEX)
	s.DefineHidden(INDEX)
	return result, nil
}

/**
* DefineAtrib: Defines the field
* @param name string, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func (s *Model) DefineAtrib(name string, tpData TypeData, defaultValue interface{}) (*Field, error) {
	return s.defineField(name, TpAtrib, tpData, defaultValue)
}

/**
* DefineDetail: Defines the detail
* @param name string, keys map[string]string, version int
* @return *Model, error
**/
func (s *Model) DefineDetail(name string, keys map[string]string, version int) (*Model, error) {
	_, err := s.defineField(name, TpDetail, TpAny, []et.Json{})
	if err != nil {
		return nil, err
	}

	to, err := newModel(s.Database, s.Schema, fmt.Sprintf("%s_%s", s.Name, name), false, version)
	if err != nil {
		return nil, err
	}

	forKeys := make(map[string]string)
	for fk, pk := range keys {
		_, err = s.DefineAtrib(pk, TpKey, "")
		if err != nil {
			return nil, err
		}

		_, err = to.DefineAtrib(fk, TpKey, "")
		if err != nil {
			return nil, err
		}

		s.DefinePrimaryKeys(pk)
		forKeys[pk] = fk
	}

	to.DefineForeignKeys(s, forKeys, true, true)
	s.Details[name] = newDetail(to.From, keys, []string{}, false, false)
	return to, nil
}

/**
* DefineRollup: Defines the rollup
* @param name string, to string, keys map[string]string, selects []string
* @return error
**/
func (s *Model) DefineRollup(name string, to *From, keys map[string]string, selects []string) error {
	_, err := s.defineField(name, TpRollup, TpJson, []et.Json{})
	if err != nil {
		return err
	}

	s.Rollups[name] = newDetail(to, keys, selects, false, false)
	return nil
}

/**
* DefineRelation: Defines the relation
* @param to *From, keys map[string]string, onDeleteCascade, onUpdateCascade bool
* @return error
**/
func (s *Model) DefineRelation(to *From, keys map[string]string, onDeleteCascade, onUpdateCascade bool) error {
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

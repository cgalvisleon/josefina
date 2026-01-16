package rds

import (
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/et"
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
	result, ok := s.Fields[name]
	if ok {
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
* defineIndexes: Defines the indexes
* @param names ...string
**/
func (s *Model) defineIndexes(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.Indexes, name)
		if idx == -1 {
			s.Indexes = append(s.Indexes, name)
		}
	}
}

/**
* defineUniques: Defines the uniques
* @param names ...string
**/
func (s *Model) defineUniques(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.Unique, name)
		if idx == -1 {
			s.Unique = append(s.Unique, name)
			s.defineIndexes(name)
		}
	}
}

/**
* defineRequireds: Defines the requireds
* @param names ...string
**/
func (s *Model) defineRequireds(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.Required, name)
		if idx == -1 {
			s.Required = append(s.Required, name)
			s.defineIndexes(name)
		}
	}
}

/**
* defineHidden: Defines the hidden
* @param names ...string
**/
func (s *Model) defineHidden(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.Hidden, name)
		if idx == -1 {
			s.Hidden = append(s.Hidden, name)
		}
	}
}

/**
* defineReferences: Defines the references
* @param names ...string
**/
func (s *Model) defineReferences(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.References, name)
		if idx == -1 {
			s.References = append(s.References, name)
			s.defineIndexes(name)
		}
	}
}

/**
* definePrimaryKeys: Defines the primary keys
* @param names ...string
**/
func (s *Model) definePrimaryKeys(names ...string) {
	for _, name := range names {
		idx := slices.Index(s.PrimaryKeys, name)
		if idx == -1 {
			s.PrimaryKeys = append(s.PrimaryKeys, name)
			s.defineRequireds(name)
			s.defineUniques(name)
		}
	}
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
	s.defineIndexes(INDEX)
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
	}

	detail := newDetail(to, s, keys, []string{}, true, true)
	s.Details[name] = detail
	fkeys := map[string]string{}
	for k, v := range keys {
		fkeys[v] = k
	}
	to.Master[s.Name] = newDetail(s, to, fkeys, []string{}, false, false)
	return to, nil
}

/**
* defineRollup: Defines the rollup
* @param name string, from string, keys map[string]string, selects []string
* @return *Model
**/
func (s *Model) defineRollup(name, from string, keys map[string]string, selects []string) error {
	_, err := s.defineFields(name, TpRollup, TpJson, []et.Json{})
	if err != nil {
		return err
	}

	to, err := s.db.getModel(s.Schema, from)
	if err != nil {
		return err
	}

	detail := newDetail(to, s, keys, selects, false, false)
	s.Rollups[name] = detail
	return nil
}

/**
* defineRelation: Defines the relation
* @param name string, from string, keys map[string]string
* @return *Model
**/
func (s *Model) defineRelation(from string, keys map[string]string) error {
	to, err := s.db.getModel(s.Schema, from)
	if err != nil {
		return err
	}

	detail := newDetail(to, s, keys, []string{}, false, false)
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

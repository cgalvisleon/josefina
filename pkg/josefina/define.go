package josefina

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
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, name)
	}

	result, ok := s.Fields[name]
	if ok {
		return result, nil
	}

	result = &Field{
		From:         s.From,
		Name:         name,
		TypeField:    tpField,
		TypeData:     tpData,
		DefaultValue: defaultValue,
		Definition:   []byte{},
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
			s.defineIndexes(name)
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

	to, err := new
}

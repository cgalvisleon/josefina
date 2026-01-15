package josefina

import "slices"

/**
* defineFields: Defines the fields
* @param name string, tpField TypeField, tpData TypeData, defaultValue interface{}
* @return *Field
**/
func (s *Model) defineFields(name string, tpField TypeField, tpData TypeData, defaultValue interface{}) *Field {
	result, ok := s.Fields[name]
	if ok {
		return result
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

	return result
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
* defineKeyField: Defines the key field
* @return *Field
**/
func (s *Model) defineKeyField() *Field {
	result := s.defineFields(KEY, TpAtrib, TpKey, "")
	s.definePrimaryKeys(KEY)
	s.defineHidden(KEY)
	return result
}

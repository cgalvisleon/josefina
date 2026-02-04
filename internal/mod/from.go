package mod

import (
	"fmt"

	"github.com/cgalvisleon/et/et"
)

type From struct {
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Name     string `json:"name"`
	isDebug  bool   `json:"-"`
}

/**
* Key: Returns the key of the model
* @return string
**/
func (s *From) Key() string {
	result := s.Name
	if s.Schema != "" {
		result = fmt.Sprintf("%s.%s", s.Schema, result)
	}
	if s.Database != "" {
		result = fmt.Sprintf("%s.%s", s.Database, result)
	}
	return result
}

/**
* ToFrom: Converts a JSON to a From
* @param def et.Json
* @return *From
**/
func ToFrom(def et.Json) *From {
	return &From{
		Database: def.Str("database"),
		Schema:   def.Str("schema"),
		Name:     def.Str("name"),
	}
}

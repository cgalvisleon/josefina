package jdb

import "github.com/cgalvisleon/et/envar"

const (
	SOURCE     string = "source"
	KEY        string = "id"
	INDEX      string = "index"
	STATUS     string = "status"
	VERSION    string = "version"
	PROJECT_ID string = "project_id"
	TENANT_ID  string = "tenant_id"
	CREATED_AT string = "created_at"
	UPDATED_AT string = "updated_at"
)

type TypeColumn string

func (s TypeColumn) Str() string {
	return string(s)
}

const (
	TpColumn      TypeColumn = "column"
	TpAtrib       TypeColumn = "atrib"
	TpDetail      TypeColumn = "detail"
	TpRollup      TypeColumn = "rollup"
	TpCalc        TypeColumn = "calc"
	TpAggregation TypeColumn = "aggregation"
)

type TypeData string

func (s TypeData) Str() string {
	return string(s)
}

const (
	TpAny      TypeData = "any"
	TpBytes    TypeData = "bytes"
	TpInt      TypeData = "int"
	TpFloat    TypeData = "float"
	TpKey      TypeData = "key"
	TpText     TypeData = "text"
	TpMemo     TypeData = "memo"
	TpJson     TypeData = "json"
	TpDateTime TypeData = "datetime"
	TpBoolean  TypeData = "boolean"
	TpGeometry TypeData = "geometry"
)

type TypeAggregation string

func (s TypeAggregation) Str() string {
	return string(s)
}

/**
* GetAggregation
* @param tp string
* @return TypeAggregation
**/
func GetAggregation(tp string) TypeAggregation {
	aggregation := map[string]TypeAggregation{
		"count": TpCount,
		"sum":   TpSum,
		"avg":   TpAvg,
		"max":   TpMax,
		"min":   TpMin,
		"exp":   TpExp,
	}

	result, ok := aggregation[tp]
	if !ok {
		return TpExp
	}
	return result
}

const (
	TpCount TypeAggregation = "count"
	TpSum   TypeAggregation = "sum"
	TpAvg   TypeAggregation = "avg"
	TpMax   TypeAggregation = "max"
	TpMin   TypeAggregation = "min"
	TpExp   TypeAggregation = "exp"
)

type Status string

const (
	Active    Status = "active"
	Archived  Status = "archived"
	Canceled  Status = "canceled"
	OfSystem  Status = "of_system"
	ForDelete Status = "for_delete"
	Pending   Status = "pending"
	Approved  Status = "approved"
	Rejected  Status = "rejected"
)

type Column struct {
	From       *Model      `json:"from"`
	Name       string      `json:"name"`
	TypeColumn TypeColumn  `json:"type_column"`
	TypeData   TypeData    `json:"type_data"`
	Default    interface{} `json:"default"`
	Definition []byte      `json:"definition"`
}

/**
* Field
* @return *Field
**/
func (s *Column) Field() *Field {
	result := &Field{
		TypeColumn: s.TypeColumn,
		Column:     s,
		Name:       s.Name,
		As:         s.Name,
	}

	if result.TypeColumn == TpAtrib {
		result.SourceField = s.From.SourceField
	} else if result.TypeColumn == TpDetail {
		if s.From == nil {
			return result
		}

		if s.From.Details == nil {
			return result
		}

		detail := s.From.Details[s.Name]
		if detail == nil {
			return result
		}

		rows := envar.GetInt("rows", 30)
		result.To = detail.To
		result.Keys = detail.Keys
		result.Select = detail.Select
		result.Page = 1
		result.Rows = rows
	} else if result.TypeColumn == TpRollup {
		if s.From == nil {
			return result
		}

		if s.From.Rollups == nil {
			return result
		}

		detail := s.From.Rollups[s.Name]
		if detail == nil {
			return result
		}

		result.To = detail.To
		result.Keys = detail.Keys
		result.Select = detail.Select
	}

	return result
}

/**
* newColumn
* @param model *Model, name string, tpColumn TypeColumn, tpData TypeData, defaultValue interface{}, definition []byte
* @return *Column
**/
func newColumn(model *Model, name string, tpColumn TypeColumn, tpData TypeData, defaultValue interface{}, definition []byte) *Column {
	return &Column{
		From:       model,
		Name:       name,
		TypeColumn: tpColumn,
		TypeData:   tpData,
		Default:    defaultValue,
		Definition: definition,
	}
}

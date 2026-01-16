package rds

import (
	"fmt"

	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

const (
	KEY        string = "id"
	INDEX      string = "_idx"
	STATUS     string = "status"
	VERSION    string = "version"
	PROJECT_ID string = "project_id"
	TENANT_ID  string = "tenant_id"
	CREATED_AT string = "created_at"
	UPDATED_AT string = "updated_at"
)

type TypeField string

func (s TypeField) Str() string {
	return string(s)
}

const (
	TpAtrib       TypeField = "atrib"
	TpDetail      TypeField = "detail"
	TpRollup      TypeField = "rollup"
	TpCalc        TypeField = "calc"
	TpAggregation TypeField = "aggregation"
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

type Field struct {
	From         *From       `json:"from"`
	Name         string      `json:"name"`
	TypeField    TypeField   `json:"type_field"`
	TypeData     TypeData    `json:"type_data"`
	DefaultValue interface{} `json:"default_value"`
	as           string      `json:"-"`
}

/**
* newField: Creates a new field
* @param from *From, name string, tpField TypeField, tpData TypeData, defaultValue interface{}
* @return *Field, error
**/
func newField(from *From, name string, tpField TypeField, tpData TypeData, defaultValue interface{}) (*Field, error) {
	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, name)
	}

	return &Field{
		From:         from,
		Name:         name,
		TypeField:    tpField,
		TypeData:     tpData,
		DefaultValue: defaultValue,
		as:           name,
	}, nil
}

/**
* clone: Clones the field
* @return *Field
**/
func (s *Field) clone() *Field {
	result, err := newField(s.From, s.Name, s.TypeField, s.TypeData, s.DefaultValue)
	if err != nil {
		return nil
	}
	return result
}

/**
* setAs
* @param as string
* @return void
**/
func (s *Field) setAs(as string) {
	s.as = as
}

/**
* As
* @return string
**/
func (s *Field) As() string {
	return fmt.Sprintf("%s.%s", s.From.As(), s.Name)
}

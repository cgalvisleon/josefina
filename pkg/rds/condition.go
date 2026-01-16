package rds

import (
	"strconv"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
)

type Operator string

const (
	OpEq         Operator = "eq"
	OpNeg        Operator = "neg"
	OpLess       Operator = "less"
	OpLessEq     Operator = "less_eq"
	OpMore       Operator = "more"
	OpMoreEq     Operator = "more_eq"
	OpLike       Operator = "like"
	OpIn         Operator = "in"
	OpNotIn      Operator = "not_in"
	OpIs         Operator = "is"
	OpIsNot      Operator = "is_not"
	OpNull       Operator = "null"
	OpNotNull    Operator = "not_null"
	OpBetween    Operator = "between"
	OpNotBetween Operator = "not_between"
)

func (s Operator) Str() string {
	return string(s)
}

func ToOperator(s string) Operator {
	values := map[string]Operator{
		"eq":          OpEq,
		"neg":         OpNeg,
		"less":        OpLess,
		"less_eq":     OpLessEq,
		"more":        OpMore,
		"more_eq":     OpMoreEq,
		"like":        OpLike,
		"in":          OpIn,
		"not_in":      OpNotIn,
		"is":          OpIs,
		"is_not":      OpIsNot,
		"null":        OpNull,
		"not_null":    OpNotNull,
		"between":     OpBetween,
		"not_between": OpNotBetween,
	}

	result, ok := values[s]
	if !ok {
		return OpEq
	}

	return result
}

type Connector string

const (
	NaC Connector = ""
	And Connector = "and"
	Or  Connector = "or"
)

func (s Connector) Str() string {
	return string(s)
}

type Condition struct {
	Field     string    `json:"field"`
	Operator  Operator  `json:"operator"`
	Value     any       `json:"value"`
	Connector Connector `json:"connector"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Condition) ToJson() et.Json {
	if s.Connector == NaC {
		return et.Json{
			s.Field: et.Json{
				s.Operator.Str(): s.Value,
			},
		}
	}

	return et.Json{
		s.Connector.Str(): et.Json{
			s.Field: et.Json{
				s.Operator.Str(): s.Value,
			},
		},
	}
}

/**
* fieldValue
* @param data et.Json
* @return any, error
**/
func (s *Condition) fieldValue(data et.Json) (any, error) {
	array := []et.Json{}
	fields := strs.Split(s.Field, ">")
	for _, field := range fields {
		idx, err := strconv.Atoi(field)
		if err == nil && len(array) > idx {
			data = array[idx]
			array = []et.Json{}
			continue
		}

		val, ok := data[field]
		if !ok {
			return nil, errorFieldNotFound
		}

		switch v := val.(type) {
		case et.Json:
			data = v
		case map[string]interface{}:
			data = v
		case []et.Json:
			array = v
		case []map[string]interface{}:
			for _, item := range v {
				array = append(array, item)
			}
		default:
			return v, nil
		}
	}

	return nil, errorFieldNotFound
}

/**
* ApplyOpEq
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpEq(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	return val == s.Value, nil
}

/**
* ApplyOpNeg
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpNeg(data et.Json) (bool, error) {
	result, err := s.ApplyOpEq(data)
	if err != nil {
		return false, err
	}

	return !result, nil
}

/**
* ApplyOpLess
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpLess(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	switch v := val.(type) {
	case int:
		return v < s.Value.(int), nil
	default:
		return false, errorInvalidType
	}
}

/**
* ApplyOpLessEq
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpLessEq(data et.Json) (bool, error) {
	val, ok := data[s.Field]
	if !ok {
		return false, errorFieldNotFound
	}

	switch v := val.(type) {
	case int:
		return v < s.Value.(int), nil
	default:
		return false, errorInvalidType
	}
}

/**
* ToCondition
* @param json et.Json
* @return *Condition
**/
func ToCondition(json et.Json) *Condition {
	getWhere := func(json et.Json) *Condition {
		for fld := range json {
			cond := json.Json(fld)
			for cnd := range cond {
				val := cond[cnd]
				return condition(fld, val, ToOperator(cnd))
			}
		}
		return nil
	}

	and := func(jsons et.Json) *Condition {
		result := getWhere(jsons)
		if result != nil {
			result.Connector = And
		}

		return result
	}

	or := func(jsons et.Json) *Condition {
		result := getWhere(jsons)
		if result != nil {
			result.Connector = Or
		}

		return result
	}

	for k := range json {
		if strs.Lowcase(k) == "and" {
			def := json.Json(k)
			return and(def)
		} else if strs.Lowcase(k) == "or" {
			def := json.Json(k)
			return or(def)
		} else {
			return getWhere(json)
		}
	}

	return nil
}

/**
* condition
* @param field string, value interface{}, op string
* @return *Condition
**/
func condition(field string, value interface{}, op Operator) *Condition {
	return &Condition{
		Field: &Field{
			Name: field,
		},
		Operator:  op,
		Value:     value,
		Connector: NaC,
	}
}

/**
* Eq
* @param field string, value interface{}
* @return Condition
**/
func Eq(field string, value interface{}) *Condition {
	return condition(field, value, OpEq)
}

/**
* Neg
* @param field string, value interface{}
* @return Condition
**/
func Neg(field string, value interface{}) *Condition {
	return condition(field, value, OpNeg)
}

/**
* Less
* @param field string, value interface{}
* @return Condition
**/
func Less(field string, value interface{}) *Condition {
	return condition(field, value, OpLess)
}

/**
* LessEq
* @param field string, value interface{}
* @return Condition
**/
func LessEq(field string, value interface{}) *Condition {
	return condition(field, value, OpLessEq)
}

/**
* More
* @param field string, value interface{}
* @return Condition
**/
func More(field string, value interface{}) *Condition {
	return condition(field, value, OpMore)
}

/**
* MoreEq
* @param field string, value interface{}
* @return Condition
**/
func MoreEq(field string, value interface{}) *Condition {
	return condition(field, value, OpMoreEq)
}

/**
* Like
* @param field string, value interface{}
* @return Condition
**/
func Like(field string, value interface{}) *Condition {
	return condition(field, value, OpLike)
}

/**
* In
* @param field string, value []interface{}
* @return Condition
**/
func In(field string, value []interface{}) *Condition {
	return condition(field, value, OpIn)
}

/**
* NotIn
* @param field string, value []interface{}
* @return Condition
**/
func NotIn(field string, value []interface{}) *Condition {
	return condition(field, value, OpNotIn)
}

/**
* Is
* @param field string, value interface{}
* @return Condition
**/
func Is(field string, value interface{}) *Condition {
	return condition(field, value, OpIs)
}

/**
* IsNot
* @param field string, value interface{}
* @return Condition
**/
func IsNot(field string, value interface{}) *Condition {
	return condition(field, value, OpIsNot)
}

/**
* Null
* @param field string
* @return Condition
**/
func Null(field string) *Condition {
	return condition(field, nil, OpNull)
}

/**
* NotNull
* @param field string
* @return Condition
**/
func NotNull(field string) *Condition {
	return condition(field, nil, OpNotNull)
}

/**
* Between
* @param field string, value []interface{}
* @return Condition
**/
func Between(field string, value []interface{}) *Condition {
	return condition(field, value, OpBetween)
}

/**
* NotBetween
* @param field string, value []interface{}
* @return Condition
**/
func NotBetween(field string, value []interface{}) *Condition {
	return condition(field, value, OpNotBetween)
}

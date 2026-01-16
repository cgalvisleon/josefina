package rds

import (
	"reflect"
	"strconv"
	"time"

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

type BetweenValue struct {
	Min any
	Max any
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

	// time.Time (soporta <)
	if av, ok := val.(time.Time); ok {
		bv, ok := s.Value.(time.Time)
		if !ok {
			return false, errorInvalidType
		}
		return av.Before(bv), nil
	}

	// string (soporta < lexicográfico)
	if av, ok := val.(string); ok {
		bv, ok := s.Value.(string)
		if !ok {
			return false, errorInvalidType
		}
		return av < bv, nil
	}

	// Números (int*, uint*, float*)
	aNum, aKind, ok := numberToFloat64(val)
	if !ok {
		return false, errorInvalidType
	}

	bNum, bKind, ok := numberToFloat64(s.Value)
	if !ok {
		return false, errorInvalidType
	}

	// Evitar comparar signed vs unsigned si hay negativos (caso peligroso)
	if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
		ai, _ := numberToInt64(val)
		if ai < 0 {
			return false, errorInvalidType
		}
	}
	if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
		bi, _ := numberToInt64(s.Value)
		if bi < 0 {
			return false, errorInvalidType
		}
	}

	return aNum < bNum, nil
}

/**
* ApplyOpLessEq
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpLessEq(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	// time.Time (soporta <=)
	if av, ok := val.(time.Time); ok {
		bv, ok := s.Value.(time.Time)
		if !ok {
			return false, errorInvalidType
		}
		return av.Before(bv) || av.Equal(bv), nil
	}

	// string (soporta <= lexicográfico)
	if av, ok := val.(string); ok {
		bv, ok := s.Value.(string)
		if !ok {
			return false, errorInvalidType
		}
		return av <= bv, nil
	}

	// Números (int*, uint*, float*)
	aNum, aKind, ok := numberToFloat64(val)
	if !ok {
		return false, errorInvalidType
	}

	bNum, bKind, ok := numberToFloat64(s.Value)
	if !ok {
		return false, errorInvalidType
	}

	// Evitar comparar signed vs unsigned si hay negativos (caso peligroso)
	if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
		ai, _ := numberToInt64(val)
		if ai < 0 {
			return false, errorInvalidType
		}
	}
	if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
		bi, _ := numberToInt64(s.Value)
		if bi < 0 {
			return false, errorInvalidType
		}
	}

	return aNum <= bNum, nil
}

/**
* ApplyOpMore
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpMore(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	// time.Time
	if av, ok := val.(time.Time); ok {
		bv, ok := s.Value.(time.Time)
		if !ok {
			return false, errorInvalidType
		}
		return av.After(bv), nil
	}

	// string
	if av, ok := val.(string); ok {
		bv, ok := s.Value.(string)
		if !ok {
			return false, errorInvalidType
		}
		return av > bv, nil
	}

	// numbers
	aNum, aKind, ok := numberToFloat64(val)
	if !ok {
		return false, errorInvalidType
	}

	bNum, bKind, ok := numberToFloat64(s.Value)
	if !ok {
		return false, errorInvalidType
	}

	// Evitar comparar signed vs unsigned si hay negativos
	if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
		ai, _ := numberToInt64(val)
		if ai < 0 {
			return false, errorInvalidType
		}
	}
	if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
		bi, _ := numberToInt64(s.Value)
		if bi < 0 {
			return false, errorInvalidType
		}
	}

	return aNum > bNum, nil
}

/**
* ApplyOpMoreEq
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpMoreEq(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	// time.Time
	if av, ok := val.(time.Time); ok {
		bv, ok := s.Value.(time.Time)
		if !ok {
			return false, errorInvalidType
		}
		return av.After(bv) || av.Equal(bv), nil
	}

	// string
	if av, ok := val.(string); ok {
		bv, ok := s.Value.(string)
		if !ok {
			return false, errorInvalidType
		}
		return av >= bv, nil
	}

	// numbers
	aNum, aKind, ok := numberToFloat64(val)
	if !ok {
		return false, errorInvalidType
	}

	bNum, bKind, ok := numberToFloat64(s.Value)
	if !ok {
		return false, errorInvalidType
	}

	// Evitar comparar signed vs unsigned si hay negativos
	if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
		ai, _ := numberToInt64(val)
		if ai < 0 {
			return false, errorInvalidType
		}
	}
	if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
		bi, _ := numberToInt64(s.Value)
		if bi < 0 {
			return false, errorInvalidType
		}
	}

	return aNum >= bNum, nil
}

/**
* ApplyOpLike
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpLike(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	av, ok := val.(string)
	if !ok {
		return false, errorInvalidType
	}

	pattern, ok := s.Value.(string)
	if !ok {
		return false, errorInvalidType
	}

	return matchLikeStar(av, pattern), nil
}

/**
* ApplyOpIn
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpIn(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	list := reflect.ValueOf(s.Value)
	if !list.IsValid() {
		return false, errorInvalidType
	}

	// Debe ser slice o array
	if list.Kind() != reflect.Slice && list.Kind() != reflect.Array {
		return false, errorInvalidType
	}

	for i := 0; i < list.Len(); i++ {
		item := list.Index(i).Interface()

		ok, err := equalsAny(val, item)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

/**
* ApplyOpNotIn
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpNotIn(data et.Json) (bool, error) {
	ok, err := s.ApplyOpIn(data)
	if err != nil {
		return false, err
	}
	return !ok, nil
}

/**
* ApplyOpIs
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpIs(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	// nil == nil
	if val == nil && s.Value == nil {
		return true, nil
	}

	// si uno es nil y el otro no -> false
	if val == nil || s.Value == nil {
		return false, nil
	}

	ok, err := equalsAny(val, s.Value)
	if err != nil {
		return false, err
	}
	return ok, nil
}

/**
* ApplyOpIsNot
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpIsNot(data et.Json) (bool, error) {
	ok, err := s.ApplyOpIs(data)
	if err != nil {
		return false, err
	}
	return !ok, nil
}

/**
* ApplyOpNull
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpNull(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}
	return val == nil, nil
}

/**
* ApplyOpNotNull
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpNotNull(data et.Json) (bool, error) {
	ok, err := s.ApplyOpNull(data)
	if err != nil {
		return false, err
	}
	return !ok, nil
}

/**
* ApplyOpBetween
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpBetween(data et.Json) (bool, error) {
	val, err := s.fieldValue(data)
	if err != nil {
		return false, err
	}

	// si el campo es nil, no puede estar entre nada
	if val == nil {
		return false, nil
	}

	min, max, ok := getBetweenRange(s.Value)
	if !ok {
		return false, errorInvalidType
	}

	// min/max no deben ser nil
	if min == nil || max == nil {
		return false, errorInvalidType
	}

	c1, ok := compareAnyOrdered(val, min) // val vs min
	if !ok {
		return false, errorInvalidType
	}
	c2, ok := compareAnyOrdered(val, max) // val vs max
	if !ok {
		return false, errorInvalidType
	}

	// BETWEEN inclusivo: val >= min && val <= max
	return c1 >= 0 && c2 <= 0, nil
}

/**
* ApplyOpNotBetween
* @param data et.Json
* @return bool, error
**/
func (s *Condition) ApplyOpNotBetween(data et.Json) (bool, error) {
	ok, err := s.ApplyOpBetween(data)
	if err != nil {
		return false, err
	}
	return !ok, nil
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
		Field:     field,
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
* @param field string, min any, max any
* @return Condition
**/
func Between(field string, min, max any) *Condition {
	return condition(field, BetweenValue{Min: min, Max: max}, OpBetween)
}

/**
* NotBetween
* @param field string, min any, max any
* @return Condition
**/
func NotBetween(field string, min, max any) *Condition {
	return condition(field, BetweenValue{Min: min, Max: max}, OpNotBetween)
}

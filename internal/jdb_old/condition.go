package jdb

import (
	"reflect"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/josefina/internal/catalog"
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
			return nil, catalog.ErrorFieldNotFound
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

	return nil, catalog.ErrorFieldNotFound
}

/**
* applyOpEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpEq(val any) bool {
	if val == nil {
		return false
	}

	switch bv := s.Value.(type) {
	case []et.Json:
		for _, item := range bv {
			for _, value := range item {
				return val == value
			}
		}
		return false
	default:
		return val == bv
	}
}

/**
* applyOpNeg
* @param val any
* @return bool
**/
func (s *Condition) applyOpNeg(val any) bool {
	result := s.applyOpEq(val)
	if result {
		return false
	}

	return !result
}

/**
* applyOpLess
* @param val any
* @return bool
**/
func (s *Condition) applyOpLess(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.Before(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av < bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum < bNum
	case []et.Json:
		for _, item := range bv {
			for _, value := range item {
				s.Value = value
				return s.applyOpLess(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			s.Value = value
			return s.applyOpLess(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpLessEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpLessEq(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.Before(bv) || av.Equal(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av <= bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum <= bNum
	case []et.Json:
		for _, item := range bv {
			for _, value := range item {
				s.Value = value
				return s.applyOpLessEq(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			s.Value = value
			return s.applyOpLessEq(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpMore
* @param val any
* @return bool
**/
func (s *Condition) applyOpMore(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.After(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av > bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum > bNum
	case []et.Json:
		for _, item := range bv {
			for _, value := range item {
				s.Value = value
				return s.applyOpMore(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			s.Value = value
			return s.applyOpMore(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpMoreEq
* @param val any
* @return bool
**/
func (s *Condition) applyOpMoreEq(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.(type) {
	case time.Time:
		if av, ok := val.(time.Time); ok {
			return av.After(bv) || av.Equal(bv)
		}
		return invalidType()
	case string:
		if av, ok := val.(string); ok {
			return av >= bv
		}
		return invalidType()
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		aNum, aKind, ok := numberToFloat64(val)
		if !ok {
			return invalidType()
		}

		bNum, bKind, ok := numberToFloat64(s.Value)
		if !ok {
			return invalidType()
		}

		if isSignedIntKind(aKind) && isUnsignedIntKind(bKind) {
			ai, _ := numberToInt64(val)
			if ai < 0 {
				return invalidType()
			}
		}
		if isUnsignedIntKind(aKind) && isSignedIntKind(bKind) {
			bi, _ := numberToInt64(s.Value)
			if bi < 0 {
				return invalidType()
			}
		}

		return aNum >= bNum
	case []et.Json:
		for _, item := range bv {
			for _, value := range item {
				s.Value = value
				return s.applyOpMoreEq(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			s.Value = value
			return s.applyOpMoreEq(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpLike
* @param val any
* @return bool
**/
func (s *Condition) applyOpLike(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	switch bv := s.Value.(type) {
	case string:
		av, ok := val.(string)
		if !ok {
			return invalidType()
		}
		return matchLikeStar(av, bv)
	case et.Json:
		for _, value := range bv {
			s.Value = value
			return s.applyOpLike(val)
		}
		return invalidType()
	case map[string]interface{}:
		for _, value := range bv {
			s.Value = value
			return s.applyOpLike(val)
		}
		return invalidType()
	case []et.Json:
		for _, item := range bv {
			for _, value := range item {
				s.Value = value
				return s.applyOpLike(val)
			}
		}
		return invalidType()
	case []interface{}:
		for _, value := range bv {
			s.Value = value
			return s.applyOpLike(val)
		}
		return invalidType()
	default:
		return invalidType()
	}
}

/**
* applyOpIn
* @param val any
* @return bool
**/
func (s *Condition) applyOpIn(val any) bool {
	if val == nil {
		return false
	}

	invalidType := func() bool {
		return false
	}

	list := reflect.ValueOf(s.Value)
	if !list.IsValid() {
		return invalidType()
	}

	if list.Kind() != reflect.Slice && list.Kind() != reflect.Array {
		return invalidType()
	}

	for i := 0; i < list.Len(); i++ {
		item := list.Index(i).Interface()

		ok, err := equalsAny(val, item)
		if err != nil {
			return false
		}
		if ok {
			return true
		}
	}

	return false
}

/**
* applyOpNotIn
* @param val any
* @return bool
**/
func (s *Condition) applyOpNotIn(val any) bool {
	ok := s.applyOpIn(val)
	return !ok
}

/**
* applyOpIs
* @param val any
* @return bool
**/
func (s *Condition) applyOpIs(val any) bool {
	if val == nil && s.Value == nil {
		return true
	}

	if val == nil || s.Value == nil {
		return false
	}

	ok, err := equalsAny(val, s.Value)
	if err != nil {
		return false
	}
	return ok
}

/**
* applyOpNull
* @param val any
* @return bool
**/
func (s *Condition) applyOpNull(val any) bool {
	return val == nil
}

/**
* applyOpNotNull
* @param val any
* @return bool
**/
func (s *Condition) applyOpNotNull(val any) bool {
	ok := s.applyOpNull(val)
	return !ok
}

/**
* applyOpBetween
* @param val any
* @return bool
**/
func (s *Condition) applyOpBetween(val any) bool {
	if val == nil {
		return false
	}

	min, max, ok := getBetweenRange(s.Value)
	if !ok {
		return false
	}

	if min == nil || max == nil {
		return false
	}

	c1, ok := compareAnyOrdered(val, min)
	if !ok {
		return false
	}

	c2, ok := compareAnyOrdered(val, max)
	if !ok {
		return false
	}

	return c1 >= 0 && c2 <= 0
}

/**
* applyOpNotBetween
* @param val any
* @return bool
**/
func (s *Condition) applyOpNotBetween(val any) bool {
	ok := s.applyOpBetween(val)
	return !ok
}

/**
* ApplyToValue
* @param val any
* @return bool
**/
func (s *Condition) ApplyToValue(val any) bool {
	switch s.Operator {
	case OpEq:
		return s.applyOpEq(val)
	case OpNeg:
		return s.applyOpNeg(val)
	case OpLess:
		return s.applyOpLess(val)
	case OpLessEq:
		return s.applyOpLessEq(val)
	case OpMore:
		return s.applyOpMore(val)
	case OpMoreEq:
		return s.applyOpMoreEq(val)
	case OpLike:
		return s.applyOpLike(val)
	case OpIn:
		return s.applyOpIn(val)
	case OpNotIn:
		return s.applyOpNotIn(val)
	case OpIs:
		return s.applyOpIs(val)
	case OpNull:
		return s.applyOpNull(val)
	case OpNotNull:
		return s.applyOpNotNull(val)
	case OpBetween:
		return s.applyOpBetween(val)
	case OpNotBetween:
		return s.applyOpNotBetween(val)
	default:
		return false
	}
}

/**
* ApplyToObject
* @param obj et.Json
* @return bool
**/
func (s *Condition) ApplyToObject(obj et.Json) bool {
	val, err := s.fieldValue(obj)
	if err != nil {
		return false
	}

	return s.ApplyToValue(val)
}

/**
* ApplyToIndex
* @param keys []string
* @return []string
**/
func (s *Condition) ApplyToIndex(keys []string) []string {
	result := make([]string, 0)
	if s.Field == "" {
		return result
	}

	for _, key := range keys {
		ok := s.ApplyToValue(key)
		if ok {
			result = append(result, key)
		}
	}

	return result
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

/**
* Validate
* @param item et.Json, conditions []*Condition
* @return bool
**/
func Validate(item et.Json, conditions []*Condition) bool {
	var result bool
	for i, con := range conditions {
		ok := con.ApplyToObject(item)
		if i == 0 {
			result = ok
			continue
		}

		if con.Connector == And {
			result = result && ok
		} else if con.Connector == Or {
			result = result || ok
		}

		if !result {
			break
		}
	}

	return result
}

package jdb

import "github.com/cgalvisleon/et/et"

/**
* Eq
* @param field string, value interface{}
* @return *et.Condition
**/
func Eq(field string, value interface{}) *et.Condition {
	return et.Eq(field, value)
}

/**
* Neg
* @param field string, value interface{}
* @return *et.Condition
**/
func Neg(field string, value interface{}) *et.Condition {
	return et.Neg(field, value)
}

/**
* Less
* @param field string, value interface{}
* @return *et.Condition
**/
func Less(field string, value interface{}) *et.Condition {
	return et.Less(field, value)
}

/**
* LessEq
* @param field string, value interface{}
* @return *et.Condition
**/
func LessEq(field string, value interface{}) *et.Condition {
	return et.LessEq(field, value)
}

/**
* More
* @param field string, value interface{}
* @return *et.Condition
**/
func More(field string, value interface{}) *et.Condition {
	return et.More(field, value)
}

/**
* MoreEq
* @param field string, value interface{}
* @return *et.Condition
**/
func MoreEq(field string, value interface{}) *et.Condition {
	return et.MoreEq(field, value)
}

/**
* Like
* @param field string, value interface{}
* @return *et.Condition
**/
func Like(field string, value interface{}) *et.Condition {
	return et.Like(field, value)
}

/**
* In
* @param field string, value []interface{}
* @return *et.Condition
**/
func In(field string, value []interface{}) *et.Condition {
	return et.In(field, value)
}

/**
* NotIn
* @param field string, value []interface{}
* @return *et.Condition
**/
func NotIn(field string, value []interface{}) *et.Condition {
	return et.NotIn(field, value)
}

/**
* Is
* @param field string, value interface{}
* @return *et.Condition
**/
func Is(field string, value interface{}) *et.Condition {
	return et.Is(field, value)
}

/**
* IsNot
* @param field string, value interface{}
* @return *et.Condition
**/
func IsNot(field string, value interface{}) *et.Condition {
	return et.IsNot(field, value)
}

/**
* Null
* @param field string
* @return *et.Condition
**/
func Null(field string) *et.Condition {
	return et.Null(field)
}

/**
* NotNull
* @param field string
* @return *et.Condition
**/
func NotNull(field string) *et.Condition {
	return et.NotNull(field)
}

/**
* Between
* @param field string, min any, max any
* @return *et.Condition
**/
func Between(field string, min, max any) *et.Condition {
	return et.Between(field, min, max)
}

/**
* NotBetween
* @param field string, min any, max any
* @return *et.Condition
**/
func NotBetween(field string, min, max any) *et.Condition {
	return et.NotBetween(field, min, max)
}

/**
* Evaluate
* @param item et.Json, wheres []*et.Condition
* @return bool
**/
func EvaluateObject(item et.Json, conditions []*et.Condition) bool {
	return et.EvaluateObject(item, conditions)
}

/**
* EvaluateValue
* @param value any, conditions []*et.Condition
* @return bool
**/
func EvaluateValue(value any, conditions []*et.Condition) bool {
	return et.EvaluateValue(value, conditions)
}

package jdb

import "github.com/cgalvisleon/josefina/jdb"

/**
* Eq
* @param field string, value interface{}
* @return jdb.Condition
**/
func Eq(field string, value interface{}) *jdb.Condition {
	return jdb.Eq(field, value)
}

/**
* Neg
* @param field string, value interface{}
* @return jdb.Condition
**/
func Neg(field string, value interface{}) *jdb.Condition {
	return jdb.Neg(field, value)
}

/**
* Less
* @param field string, value interface{}
* @return jdb.Condition
**/
func Less(field string, value interface{}) *jdb.Condition {
	return jdb.Less(field, value)
}

/**
* LessEq
* @param field string, value interface{}
* @return jdb.Condition
**/
func LessEq(field string, value interface{}) *jdb.Condition {
	return jdb.LessEq(field, value)
}

/**
* More
* @param field string, value interface{}
* @return jdb.Condition
**/
func More(field string, value interface{}) *jdb.Condition {
	return jdb.More(field, value)
}

/**
* MoreEq
* @param field string, value interface{}
* @return jdb.Condition
**/
func MoreEq(field string, value interface{}) *jdb.Condition {
	return jdb.MoreEq(field, value)
}

/**
* Like
* @param field string, value interface{}
* @return jdb.Condition
**/
func Like(field string, value interface{}) *jdb.Condition {
	return jdb.Like(field, value)
}

/**
* In
* @param field string, value []interface{}
* @return jdb.Condition
**/
func In(field string, value []interface{}) *jdb.Condition {
	return jdb.In(field, value)
}

/**
* NotIn
* @param field string, value []interface{}
* @return jdb.Condition
**/
func NotIn(field string, value []interface{}) *jdb.Condition {
	return jdb.NotIn(field, value)
}

/**
* Is
* @param field string, value []interface{}
* @return jdb.Condition
**/
func Is(field string, value []interface{}) *jdb.Condition {
	return jdb.Is(field, value)
}

/**
* IsNot
* @param field string, value []interface{}
* @return jdb.Condition
**/
func IsNot(field string, value []interface{}) *jdb.Condition {
	return jdb.IsNot(field, value)
}

/**
* Null
* @param field string
* @return jdb.Condition
**/
func Null(field string) *jdb.Condition {
	return jdb.Null(field)
}

/**
* NotNull
* @param field string
* @return jdb.Condition
**/
func NotNull(field string) *jdb.Condition {
	return jdb.NotNull(field)
}

/**
* Between
* @param field string, value []interface{}
* @return jdb.Condition
**/
func Between(field string, value []interface{}) *jdb.Condition {
	return jdb.Between(field, value)
}

/**
* NotBetween
* @param field string, value []interface{}
* @return jdb.Condition
**/
func NotBetween(field string, value []interface{}) *jdb.Condition {
	return jdb.NotBetween(field, value)
}

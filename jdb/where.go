package jdb

import "github.com/cgalvisleon/et/et"

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

type Connector string

const (
	And Connector = "and"
	Or  Connector = "or"
)

func (s Connector) Str() string {
	return string(s)
}

type Condition struct {
	Field     *Field    `json:"field"`
	Operator  Operator  `json:"operator"`
	Value     any       `json:"value"`
	Connector Connector `json:"connector"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Condition) ToJson() et.Json {
	return et.Json{
		s.Field.AS(): et.Json{
			s.Operator.Str(): s.Value,
		},
	}
}

type Wheres struct {
	Owner      interface{}  `json:"-"`
	Conditions []*Condition `json:"conditions"`
}

func (s *Wheres) ToJson() []et.Json {
	result := []et.Json{}
	and := []et.Json{}
	or := []et.Json{}
	for i, condition := range s.Conditions {
		if i == 0 {
			result = append(result, condition.ToJson())
		} else if condition.Connector == Or {
			or = append(or, condition.ToJson())
		} else {
			and = append(and, condition.ToJson())
		}
	}

	if len(and) > 0 {
		result = append(result, et.Json{
			And.Str(): and,
		})
	}

	if len(or) > 0 {
		result = append(result, et.Json{
			Or.Str(): or,
		})
	}

	return result
}

/**
* newWhere
* @param owner interface{}
* @return *Wheres
**/
func newWhere(owner interface{}) *Wheres {
	return &Wheres{
		Owner:      owner,
		Conditions: make([]*Condition, 0),
	}
}

/**
* Add
* @param condition *Condition
* @return void
**/
func (s *Wheres) Add(condition *Condition) {
	switch v := s.Owner.(type) {
	case *Cmd:
		condition.Field = v.Model.FindField(condition.Field.Name)
		condition.Connector = And
	case *Ql:
		condition.Field = FindField(v.Froms, condition.Field.Name)
		condition.Connector = And
	}

	s.Conditions = append(s.Conditions, condition)
}

/**
* Or
* @param condition *Condition
* @return void
**/
func (s *Wheres) Or(condition *Condition) {
	condition.Connector = Or
	s.Add(condition)
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
		Connector: And,
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

package jdb

import (
	"strings"

	"github.com/cgalvisleon/et/et"
)

type JoinType int

const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
	FullJoin
)

/**
* String
* @return string
**/
func (j JoinType) String() string {
	switch j {
	case InnerJoin:
		return "inner"
	case LeftJoin:
		return "left"
	case RightJoin:
		return "right"
	case FullJoin:
		return "full"
	default:
		return ""
	}
}

type Join struct {
	To   *Model            `json:"to"`
	Keys map[string]string `json:"keys"`
	Type JoinType          `json:"type"`
}

type OrderField struct {
	Field string
	Asc   bool
}

/**
* Query
**/
type Query struct {
	db      *DB             `json:"-"`
	from    *Model          `json:"-"`
	joins   []Join          `json:"-"`
	selects []string        `json:"-"`
	hidden  []string        `json:"-"`
	wheres  []*et.Condition `json:"-"`
	groupBy []string        `json:"-"`
	orderBy []OrderField    `json:"-"`
	having  []*et.Condition `json:"-"`
	offset  int             `json:"-"`
	limit   int             `json:"-"`
	isDebug bool            `json:"-"`
}

/**
* newWhere
* @param owner *Model
* @return *Query
**/
func newWhere(model *Model) *Query {
	result := &Query{
		db:      model.db,
		from:    model,
		joins:   make([]Join, 0),
		selects: make([]string, 0),
		hidden:  make([]string, 0),
		wheres:  make([]*et.Condition, 0),
		groupBy: make([]string, 0),
		orderBy: make([]OrderField, 0, 2),
		having:  make([]*et.Condition, 0),
		offset:  0,
		limit:   0,
		isDebug: false,
	}

	return result
}

/**
* IsDebug: Returns the debug mode
* @return *Query
**/
func (s *Query) IsDebug() *Query {
	s.isDebug = true
	return s
}

/**
* ToJson
* @return et.Json
**/
func (s *Query) ToJson() []et.Json {
	result := []et.Json{}
	for _, condition := range s.wheres {
		result = append(result, condition.ToJson())
	}
	return result
}

/**
* From
* @param model *Model
* @return *Query
**/
func (s *Query) From(model *Model) *Query {
	if model == nil {
		return s
	}

	s.from = model
	return s
}

/**
* Add
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) Add(condition *et.Condition) *Query {
	if len(s.wheres) > 0 && condition.Connector == et.NaC {
		condition.Connector = et.And
	}

	s.wheres = append(s.wheres, condition)
	return s
}

/**
* Query
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) Query(condition *et.Condition) *Query {
	return s.Add(condition)
}

/**
* And
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) And(condition *et.Condition) *Query {
	condition.Connector = et.And
	return s.Add(condition)
}

/**
* Or
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) Or(condition *et.Condition) *Query {
	condition.Connector = et.Or
	return s.Add(condition)
}

/**
* Selects
* @param fields ...string
* @return *Query
**/
func (s *Query) Selects(fields ...string) *Query {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.selects = append(s.selects, field)
	}

	return s
}

/**
* Hidden
* @param fields ...string
* @return *Query
**/
func (s *Query) Hidden(fields ...string) *Query {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.hidden = append(s.hidden, field)
	}

	return s
}

/**
* Asc
* @param field string
* @return *Query
**/
func (s *Query) Asc(field string) *Query {
	s.orderBy = append(s.orderBy, OrderField{Field: strings.ToLower(field), Asc: true})
	return s
}

/**
* Desc
* @param field string
* @return *Query
**/
func (s *Query) Desc(field string) *Query {
	s.orderBy = append(s.orderBy, OrderField{Field: strings.ToLower(field), Asc: false})
	return s
}

/**
* Order
* @param field string
* @return bool
**/
func (s *Query) Order(field string) bool {
	field = strings.ToLower(field)
	for _, ob := range s.orderBy {
		if ob.Field == field {
			return ob.Asc
		}
	}
	return true
}

/**
* InnerJoin: Adds an inner join — only primary records with a match in to are returned.
* @param to *Model, keys map[string]string
* @return *Query
**/
func (s *Query) InnerJoin(to *Model, keys map[string]string) *Query {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: InnerJoin})
	return s
}

/**
* LeftJoin: Adds a left join — all primary records are returned; joined fields empty when no match.
* @param to *Model, keys map[string]string
* @return *Query
**/
func (s *Query) LeftJoin(to *Model, keys map[string]string) *Query {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: LeftJoin})
	return s
}

/**
* RightJoin: Adds a right join — all records from to are returned; primary fields empty when no match.
* @param to *Model, keys map[string]string
* @return *Query
**/
func (s *Query) RightJoin(to *Model, keys map[string]string) *Query {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: RightJoin})
	return s
}

/**
* FullJoin: Adds a full join — all records from both models, matched where possible.
* @param to *Model, keys map[string]string
* @return *Query
**/
func (s *Query) FullJoin(to *Model, keys map[string]string) *Query {
	s.joins = append(s.joins, Join{To: to, Keys: keys, Type: FullJoin})
	return s
}

/**
* Limit
* @param page int, rows int
* @return *Query
**/
func (s *Query) Limit(page int, rows int) *Query {
	offset := (page - 1) * rows
	s.offset = offset
	s.limit = rows
	return s
}

/**
* SetOffset: Sets raw offset and row limit without page arithmetic.
* @param offset int, rows int
* @return *Query
**/
func (s *Query) SetOffset(offset int, rows int) *Query {
	s.offset = offset
	s.limit = rows
	return s
}

/**
* From
* @param model *Model
* @return *Query
**/
func From(model *Model) *Query {
	return newWhere(model)
}

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
func Evaluate(item et.Json, conditions []*et.Condition) bool {
	return et.Evaluate(item, conditions)
}

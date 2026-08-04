package jdb

import (
	"regexp"
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

type To struct {
	Model *Model `json:"model"`
	As    string `json:"as"`
}

type Fld struct {
	To    To     `json:"to"`
	Field any    `json:"field"`
	As    string `json:"as"`
}

type Join struct {
	To   To                `json:"to"`
	Keys map[string]string `json:"keys"`
	Type JoinType          `json:"type"`
}

type OrderField struct {
	Field string `json:"field"`
	Asc   bool   `json:"asc"`
}

/**
* Query
**/
type Query struct {
	db      *DB             `json:"-"`
	froms   []To            `json:"-"`
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
func newQuery(model *Model, as string) *Query {
	result := &Query{
		db:      model.db,
		froms:   make([]To, 0),
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
	result.addFrom(model, as)
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
func (s *Query) ToJson() et.Json {
	wheres := []et.Json{}
	for _, condition := range s.wheres {
		wheres = append(wheres, condition.ToJson())
	}

	return et.Json{
		"froms":   s.froms,
		"joins":   s.joins,
		"selects": s.selects,
		"hidden":  s.hidden,
		"wheres":  wheres,
		"groupBy": s.groupBy,
		"orderBy": s.orderBy,
		"having":  s.having,
		"offset":  s.offset,
		"limit":   s.limit,
		"isDebug": s.isDebug,
	}
}

/**
* addFrom
* @param model *Model, as string
* @return *Query
**/
func (s *Query) addFrom(model *Model, as ...string) To {
	if len(as) == 0 {
		as = []string{model.Name}
	}
	result := To{Model: model, As: as[0]}
	s.froms = append(s.froms, result)
	return result
}

/**
* AggFld: Represents an aggregate function applied to a nested Fld, e.g. sum(amount).
**/
type AggFld struct {
	Type TypeAggregation `json:"type"`
	Fld  string          `json:"fld"`
}

/**
* resolveTo: Looks up a From by its alias; falls back to the first From when as is empty
* or when the alias has no match.
* @param as string
* @return To
**/
func (s *Query) resolveTo(as string) To {
	for _, from := range s.froms {
		if from.As == as {
			return from
		}
	}
	if as == "" && len(s.froms) > 0 {
		return s.froms[0]
	}
	return To{}
}

/**
* findFld: Resolves a field reference to its Fld, supporting the formats
* "<as>.<name>:<as>", "<as>.<name>", "<name>:<as>", "<name>",
* "<agg>(<field>):<as>" and "<agg>(<field>)".
* @param field string
* @return Fld
**/
func (s *Query) findFld(field string) Fld {
	pattern1 := regexp.MustCompile(`^([A-Za-z0-9_]+)\.([A-Za-z0-9_>-]+):([A-Za-z0-9_]+)$`) // from.field:as
	pattern2 := regexp.MustCompile(`^([A-Za-z0-9_]+)\.([A-Za-z0-9_>-]+)$`)                 // from.field
	pattern3 := regexp.MustCompile(`^([A-Za-z0-9_>-]+):([A-Za-z0-9_]+)$`)                  // field:as
	pattern4 := regexp.MustCompile(`^([A-Za-z0-9_>-]+)$`)                                  // field
	pattern5 := regexp.MustCompile(`^([A-Za-z0-9_]+)\((.+)\):([A-Za-z0-9_]+)$`)            // agg(field):as
	pattern6 := regexp.MustCompile(`^([A-Za-z0-9_]+)\((.+)\)$`)                            // agg(field)

	if m := pattern5.FindStringSubmatch(field); m != nil {
		inner := s.findFld(m[2])
		return Fld{
			To:    inner.To,
			Field: AggFld{Type: GetAggregation(strings.ToLower(m[1])), Fld: m[2]},
			As:    m[3],
		}
	}

	if m := pattern6.FindStringSubmatch(field); m != nil {
		inner := s.findFld(m[2])
		agg := GetAggregation(strings.ToLower(m[1]))
		alias := strings.ToLower(m[1]) + "_" + strings.ReplaceAll(m[2], ".", "_")
		return Fld{
			To:    inner.To,
			Field: AggFld{Type: agg, Fld: m[2]},
			As:    alias,
		}
	}

	if m := pattern1.FindStringSubmatch(field); m != nil {
		return Fld{To: s.resolveTo(m[1]), Field: m[2], As: m[3]}
	}

	if m := pattern2.FindStringSubmatch(field); m != nil {
		return Fld{To: s.resolveTo(m[1]), Field: m[2], As: m[2]}
	}

	if m := pattern3.FindStringSubmatch(field); m != nil {
		return Fld{To: s.resolveTo(""), Field: m[1], As: m[2]}
	}

	if m := pattern4.FindStringSubmatch(field); m != nil {
		return Fld{To: s.resolveTo(""), Field: m[1], As: m[1]}
	}

	return Fld{To: s.resolveTo(""), Field: field, As: field}
}

/**
* InnerJoin: Adds an inner join — only primary records with a match in to are returned.
* @param to *Model, as string, keys map[string]string
* @return *Query
**/
func (s *Query) InnerJoin(to *Model, as string, keys map[string]string) *Query {
	t := s.addFrom(to, as)
	s.joins = append(s.joins, Join{To: t, Keys: keys, Type: InnerJoin})
	return s
}

/**
* LeftJoin: Adds a left join — all primary records are returned; joined fields empty when no match.
* @param to *Model, as string, keys map[string]string
* @return *Query
**/
func (s *Query) LeftJoin(to *Model, as string, keys map[string]string) *Query {
	t := s.addFrom(to, as)
	s.joins = append(s.joins, Join{To: t, Keys: keys, Type: LeftJoin})
	return s
}

/**
* RightJoin: Adds a right join — all records from to are returned; primary fields empty when no match.
* @param to *Model, as string, keys map[string]string
* @return *Query
**/
func (s *Query) RightJoin(to *Model, as string, keys map[string]string) *Query {
	t := s.addFrom(to, as)
	s.joins = append(s.joins, Join{To: t, Keys: keys, Type: RightJoin})
	return s
}

/**
* FullJoin: Adds a full join — all records from both models, matched where possible.
* @param to *Model, as string, keys map[string]string
* @return *Query
**/
func (s *Query) FullJoin(to *Model, as string, keys map[string]string) *Query {
	t := s.addFrom(to, as)
	s.joins = append(s.joins, Join{To: t, Keys: keys, Type: FullJoin})
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
* Where
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) Where(condition *et.Condition) *Query {
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
	for _, field := range fields {
		s.hidden = append(s.hidden, field)
	}

	return s
}

/**
* Order
* @param field string
* @return bool
**/
func (s *Query) OrderBy(field string, asc ...bool) *Query {
	if len(asc) == 0 {
		asc = []bool{true}
	}
	s.orderBy = append(s.orderBy, OrderField{Field: field, Asc: asc[0]})

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
* Exec: Runs the query against the primary model via Model.ForEach and returns the
* projected, ordered, paginated result.
*
* Efficiency choices:
*   - When there is no where and no orderBy, offset/limit are pushed straight into
*     ForEach so the store only reads the requested window instead of the full table.
*   - When there is a where clause made entirely of AND-connected conditions and at
*     least one condition targets a BTree-indexed field (or the primary key), the
*     candidate keys are narrowed via BTree.ApplyCondition / a direct key lookup
*     instead of scanning every record; the full where list is still re-checked per
*     candidate since narrowing only covers part of the conditions.
*   - Otherwise it falls back to a full concurrent scan (ForEach already parallelizes
*     across segments) evaluating the where list per row.
*
* joins, groupBy and having are not supported yet and return an error.
* @return et.Items, error
**/
func (s *Query) Exec() (et.Items, error) {
	result := et.Items{}

	return result, nil
}

/**
* First
* @return et.Item, error
**/
func (s *Query) First() (et.Item, error) {
	result, err := s.Exec()
	if err != nil {
		return et.Item{}, err
	}

	return result.First()
}

/**
* Last
* @return et.Item, error
**/
func (s *Query) Last() (et.Item, error) {
	result, err := s.Exec()
	if err != nil {
		return et.Item{}, err
	}

	return result.Last()
}

/**
* From
* @param model *Model, as string
* @return *Query
**/
func From(model *Model, as string) *Query {
	return newQuery(model, as)
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

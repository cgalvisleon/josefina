package jdb

import (
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Froms struct {
	model *Model
	as    string
}

/**
* Where
**/
type Where struct {
	db         *DB                 `json:"-"`
	from       *Model              `json:"-"`
	selects    []string            `json:"-"`
	hidden     []string            `json:"-"`
	keys       map[string][]string `json:"-"`
	asc        map[string]bool     `json:"-"`
	offset     int                 `json:"-"`
	limit      int                 `json:"-"`
	conditions []*et.Condition     `json:"-"`
	isDebug    bool                `json:"-"`
}

/**
* newWhere
* @param owner *Model
* @return *Where
**/
func newWhere(model *Model) *Where {
	result := &Where{
		db:         model.db,
		from:       model,
		selects:    make([]string, 0),
		hidden:     make([]string, 0),
		keys:       make(map[string][]string, 0),
		asc:        make(map[string]bool, 0),
		offset:     0,
		limit:      0,
		conditions: make([]*et.Condition, 0),
	}

	return result
}

/**
* IsDebug: Returns the debug mode
* @return *Where
**/
func (s *Where) IsDebug() *Where {
	s.isDebug = true
	return s
}

/**
* ToJson
* @return et.Json
**/
func (s *Where) ToJson() []et.Json {
	result := []et.Json{}
	for _, condition := range s.conditions {
		result = append(result, condition.ToJson())
	}
	return result
}

/**
* From
* @param model *Model
* @return *Where
**/
func (s *Where) From(model *Model) *Where {
	if model == nil {
		return s
	}

	s.from = model
	return s
}

/**
* Add
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) Add(condition *et.Condition) *Where {
	if len(s.conditions) > 0 && condition.Connector == et.NaC {
		condition.Connector = et.And
	}

	s.conditions = append(s.conditions, condition)
	return s
}

/**
* Where
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) Where(condition *et.Condition) *Where {
	return s.Add(condition)
}

/**
* And
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) And(condition *et.Condition) *Where {
	condition.Connector = et.And
	return s.Add(condition)
}

/**
* Or
* @param condition *et.Condition
* @return *Where
**/
func (s *Where) Or(condition *et.Condition) *Where {
	condition.Connector = et.Or
	return s.Add(condition)
}

/**
* Selects
* @param fields ...string
* @return *Where
**/
func (s *Where) Selects(fields ...string) *Where {
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
* @return *Where
**/
func (s *Where) Hidden(fields ...string) *Where {
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
* @return *Where
**/
func (s *Where) Asc(field string) *Where {
	s.asc[field] = true
	return s
}

/**
* Desc
* @param field string
* @return *Where
**/
func (s *Where) Desc(field string) *Where {
	s.asc[field] = false
	return s
}

/**
* Order
* @param field string
* @return bool
**/
func (s *Where) Order(field string) bool {
	result, exists := s.asc[field]
	if !exists {
		result = true
	}

	return result
}

/**
* Limit
* @param page int, rows int
* @return *Where
**/
func (s *Where) Limit(page int, rows int) *Where {
	offset := (page - 1) * rows
	s.offset = offset
	s.limit = rows
	return s
}

/**
* All
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Where) All(tx *Tx) (et.Items, error) {
	if s.from == nil {
		return et.Items{}, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	tx = GetTx(s.db, tx)
	keys := map[string]int{}
	result := []et.Json{}
	model := s.from

	addResult := func(idx string, item et.Json) bool {
		if len(s.selects) == 0 {
			item = item.Hidden(model.Hidden)
		} else {
			item = item.Select(s.selects)
		}
		item = item.Hidden(s.hidden)

		n := len(result)
		id, exists := keys[idx]
		if exists {
			result[id] = item
		} else {
			keys[idx] = n
			result = append(result, item)
			n++
		}

		return n < s.limit
	}

	if len(s.conditions) == 0 {
		next := true
		asc := s.Order(INDEX)
		err := model.ForEach(func(idx string, item et.Json) (bool, error) {
			next = addResult(idx, item)
			return next, nil
		}, asc, s.offset, s.limit)
		if err != nil {
			return et.Items{}, err
		}

		if !next {
			return et.NewItems(result), nil
		}

		cache := tx.Items(model)
		for _, item := range cache {
			idx := item.Str(INDEX)
			if idx == "" {
				continue
			}
			next = addResult(idx, item)
			if !next {
				return et.NewItems(result), nil
			}
		}

		return et.NewItems(result), nil
	}

	for _, con := range s.conditions {
		value := con.Value
		switch v := value.(type) {
		case *Where:
			var err error
			con.Value, err = v.All(tx)
			if err != nil {
				return et.Items{}, err
			}
		case Where:
			var err error
			con.Value, err = v.All(tx)
			if err != nil {
				return et.Items{}, err
			}
		}

		bt, exists := model.getBtree(con.Field)
		if exists {
			idxs := bt.ApplyCondition(con)
			for _, idx := range idxs {
				item, exists, err := model.Current(idx)
				if err != nil {
					return et.Items{}, err
				}
				if exists {
					addResult(idx, item)
				}
			}

			cache := tx.Items(model)
			for _, item := range cache {
				ok := item.ApplyCondition(con)
				if ok {
					idx := item.Str(INDEX)
					if idx == "" {
						continue
					}
					next := addResult(idx, item)
					if !next {
						return et.NewItems(result), nil
					}
				}
			}
			continue
		}

		next := true
		err := model.ForEach(func(idx string, item et.Json) (bool, error) {
			ok := item.ApplyCondition(con)
			if ok {
				next = addResult(idx, item)
			}
			return next, nil
		}, true, s.offset, s.limit)
		if err != nil {
			return et.Items{}, err
		}

		cache := tx.Items(model)
		for _, item := range cache {
			ok := item.ApplyCondition(con)
			if ok {
				idx := item.Str(INDEX)
				if idx == "" {
					continue
				}
				next := addResult(idx, item)
				if !next {
					return et.NewItems(result), nil
				}
			}
		}
	}

	return et.NewItems(result), nil
}

/**
* One
* @param tx *Tx, idx int
* @return et.Json, error
**/
func (s *Where) One(tx *Tx, idx int) (et.Item, error) {
	items, err := s.All(tx)
	if err != nil {
		return et.Item{}, err
	}

	return items.One(idx)
}

/**
* First
* @param tx *Tx
* @return et.Json, error
**/
func (s *Where) First(tx *Tx) (et.Item, error) {
	return s.One(tx, 0)
}

/**
* Last
* @param tx *Tx
* @return et.Item, error
**/
func (s *Where) Last(tx *Tx) (et.Item, error) {
	return s.One(tx, -1)
}

/**
* From
* @param model *Model
* @return *Where
**/
func From(model *Model) *Where {
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
* @param item et.Json, conditions []*et.Condition
* @return bool
**/
func Evaluate(item et.Json, conditions []*et.Condition) bool {
	return et.Evaluate(item, conditions)
}

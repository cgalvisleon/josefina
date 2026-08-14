package jdb

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type OrderField struct {
	Field string `json:"field"`
	Asc   bool   `json:"asc"`
}

type Where struct {
	model      *Model                     `json:"-"`
	selects    []string                   `json:"-"`
	hidden     []string                   `json:"-"`
	conditions map[string][]*et.Condition `json:"-"`
	orderBy    []OrderField               `json:"-"`
	offset     int                        `json:"-"`
	limit      int                        `json:"-"`
	isDebug    bool                       `json:"-"`
}

/**
* newWhere
* @param model *Model
* @return *Where
**/
func newWhere(model *Model) *Where {
	limit := envar.GetInt("JDB_LIMIT", 100)
	result := &Where{
		model:      model,
		selects:    make([]string, 0),
		hidden:     make([]string, 0),
		conditions: make(map[string][]*et.Condition),
		orderBy:    make([]OrderField, 0),
		offset:     0,
		limit:      limit,
		isDebug:    false,
	}
	return result
}

/**
* ToJson
* @return et.Json
**/
func (s *Where) ToJson() et.Json {
	return et.Json{
		"model":      s.model,
		"selects":    s.selects,
		"hidden":     s.hidden,
		"conditions": s.conditions,
		"orderBy":    s.orderBy,
		"offset":     s.offset,
		"limit":      s.limit,
	}
}

/**
* IsDebug
* @return *Where
**/
func (s *Where) IsDebug() *Where {
	s.isDebug = true
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

	key := condition.Key()
	s.conditions[key] = append(s.conditions[key], condition)
	return s
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
	for _, field := range fields {
		s.hidden = append(s.hidden, field)
	}
	return s
}

/**
* OrderBy
* @param field string
* @return *Where
**/
func (s *Where) OrderBy(field string, asc ...bool) *Where {
	if len(asc) == 0 {
		asc = []bool{true}
	}
	s.orderBy = append(s.orderBy, OrderField{Field: field, Asc: asc[0]})
	return s
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

type Query struct {
	wheres  []*Where        `json:"-"`
	groupBy []string        `json:"-"`
	having  []*et.Condition `json:"-"`
	active  *Where          `json:"-"`
	isDebug bool            `json:"-"`
}

/**
* newQuery
* @param model *Model
* @return *Query
**/
func newQuery(model *Model) *Query {
	result := &Query{
		wheres:  make([]*Where, 0),
		groupBy: make([]string, 0),
		having:  make([]*et.Condition, 0),
		isDebug: false,
	}
	where := newWhere(model)
	result.addWhere(where)
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
	for _, where := range s.wheres {
		wheres = append(wheres, where.ToJson())
	}

	return et.Json{
		"wheres":  s.wheres,
		"groupBy": s.groupBy,
		"having":  s.having,
	}
}

/**
* addFrom
* @param where *Where
* @return *Query
**/
func (s *Query) addWhere(where *Where) *Query {
	s.active = where
	s.wheres = append(s.wheres, s.active)
	return s
}

/**
* add
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) add(condition *et.Condition) *Query {
	if len(s.wheres) > 0 && condition.Connector == et.NaC {
		condition.Connector = et.And
	}

	s.active.Add(condition)
	return s
}

/**
* Where
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) Where(condition *et.Condition) *Query {
	return s.add(condition)
}

/**
* And
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) And(condition *et.Condition) *Query {
	condition.Connector = et.And
	return s.add(condition)
}

/**
* Or
* @param condition *et.Condition
* @return *Query
**/
func (s *Query) Or(condition *et.Condition) *Query {
	condition.Connector = et.Or
	return s.add(condition)
}

/**
* Selects
* @param fields ...string
* @return *Query
**/
func (s *Query) Selects(fields ...string) *Query {
	s.active.Selects(fields...)
	return s
}

/**
* Hidden
* @param fields ...string
* @return *Query
**/
func (s *Query) Hidden(fields ...string) *Query {
	s.active.Hidden(fields...)
	return s
}

/**
* Order
* @param field string, asc ...bool
* @return bool
**/
func (s *Query) OrderBy(field string, asc ...bool) *Query {
	s.active.OrderBy(field, asc...)
	return s
}

/**
* Limit
* @param page int, rows int
* @return *Query
**/
func (s *Query) Limit(page int, rows int) *Query {
	s.active.Limit(page, rows)
	return s
}

/**
* Exec: Runs the query against the primary model via Model.ForEach and returns there result.
* @param model Source, offset int, limit int
* @return et.Items, error
**/
func (s *Query) Exec() (et.Items, error) {
	result := et.Items{}
	otherConditions := []*et.Condition{}
	for _, where := range s.wheres {
		model := where.model
		for name, conditions := range where.conditions {
			if name == INDEX {
				store, exists := model.source()
				if exists {
					var err error
					result, err = store.EvaluateValue(conditions, result)
					if err != nil {
						return result, err
					}
					continue
				}
			}

			bTree, exists := model.getBtree(name)
			if exists {
				var err error
				keys := bTree.ApplyConditions(conditions)
				if err != nil {
					return result, err
				}
				for _, key := range keys {
					item, err := model.getObject(key)
					if err != nil {
						return result, err
					}
					result.Add(item)
				}
				continue
			}

			otherConditions = append(otherConditions, conditions...)
		}

		model.ForEach(func(idx string, item et.Json) (bool, error) {
			ok := et.EvaluateObject(item, otherConditions)
			if ok {
				result.Add(item)
			}
			return ok, nil
		}, true, where.offset, where.limit)
	}

	return result, nil
}

/**
* One
* @return et.Item, error
**/
func (s *Query) One() (et.Item, error) {
	result, err := s.Exec()
	if err != nil {
		return et.Item{}, err
	}

	return result.First()
}

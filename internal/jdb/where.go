package jdb

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/internal/catalog"
	"github.com/cgalvisleon/josefina/internal/msg"
)

type Froms struct {
	model *catalog.Model
	as    string
}

/**
* Where
**/
type Where struct {
	from       *catalog.Model      `json:"-"`
	selects    []string            `json:"-"`
	hidden     []string            `json:"-"`
	keys       map[string][]string `json:"-"`
	asc        map[string]bool     `json:"-"`
	offset     int                 `json:"-"`
	limit      int                 `json:"-"`
	conditions []*Condition        `json:"-"`
	workers    int                 `json:"-"`
	isDebug    bool                `json:"-"`
}

/**
* newWhere
* @param owner *Model
* @return *Where
**/
func newWhere(model *catalog.Model) *Where {
	result := &Where{
		from:       model,
		selects:    make([]string, 0),
		hidden:     make([]string, 0),
		keys:       make(map[string][]string, 0),
		asc:        make(map[string]bool, 0),
		offset:     0,
		limit:      0,
		conditions: make([]*Condition, 0),
		workers:    1,
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
* From
* @param model *Model
* @return *Where
**/
func (s *Where) From(model *catalog.Model) *Where {
	if model == nil {
		return s
	}

	s.from = model
	return s
}

/**
* ToJson
* @return et.Json
**/
func (s *Where) ToJson() (et.Json, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Add
* @param condition *Condition
* @return *Where
**/
func (s *Where) Add(condition *Condition) *Where {
	if len(s.conditions) > 0 && condition.Connector == NaC {
		condition.Connector = And
	}

	s.conditions = append(s.conditions, condition)
	return s
}

/**
* Where
* @param condition *Condition
* @return *Where
**/
func (s *Where) Where(condition *Condition) *Where {
	return s.Add(condition)
}

/**
* And
* @param condition *Condition
* @return *Where
**/
func (s *Where) And(condition *Condition) *Where {
	condition.Connector = And
	return s.Add(condition)
}

/**
* Or
* @param condition *Condition
* @return *Where
**/
func (s *Where) Or(condition *Condition) *Where {
	condition.Connector = Or
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
	s.limit = rows
	s.offset = offset
	return s
}

/**
* Run
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Where) Run(tx *Tx) ([]et.Json, error) {
	if s.from == nil {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	tx, _ = GetTx(tx)
	result := []et.Json{}
	model := s.from

	addResult := func(item et.Json) bool {
		if len(s.selects) == 0 {
			item = hidden(model.Hidden, item)
		} else {
			item = selects(s.selects, item)
			item = hidden(model.Hidden, item)
		}
		result = append(result, item)
		n := len(result)
		return n < s.limit
	}

	if len(s.conditions) == 0 {
		next := true
		asc := s.Order(catalog.INDEX)
		err := model.For(func(idx string, item et.Json) (bool, error) {
			next = addResult(item)
			return next, nil
		}, asc, s.offset, s.limit, s.workers)
		if err != nil {
			return nil, err
		}

		if !next {
			return result, nil
		}

		cache := tx.getCache(model.From)
		for _, item := range cache {
			next = addResult(item)
			if !next {
				return result, nil
			}
		}

		return result, nil
	}

	onlyKeys := true
	for _, con := range s.conditions {
		value := con.Value
		switch v := value.(type) {
		case *Where:
			var err error
			con.Value, err = v.Run(tx)
			if err != nil {
				return nil, err
			}
		case Where:
			var err error
			con.Value, err = v.Run(tx)
			if err != nil {
				return nil, err
			}
		}

		field := con.Field
		index, err := model.Store(field)
		if err != nil {
			onlyKeys = false
			continue
		}

		keys, ok := s.keys[field]
		if !ok {
			asc := s.Order(field)
			keys = index.Keys(asc, 0, 0)
		}

		s.keys[field] = con.ApplyToIndex(keys)
	}

	// Items by keys
	items := []et.Json{}
	addItem := func(item et.Json) {
		index, ok := item[catalog.INDEX]
		if !ok {
			return
		}

		if index == "" {
			return
		}

		idx := slices.IndexFunc(items, func(v et.Json) bool { return v[catalog.INDEX] == index })
		if idx == -1 {
			items = append(items, item)
		}
	}

	for field, keys := range s.keys {
		for _, key := range keys {
			indexes := map[string]bool{}
			exists, err := model.GetIndex(field, key, indexes)
			if err != nil {
				return nil, err
			}
			if !exists {
				continue
			}

			for idx := range indexes {
				item := et.Json{}
				exists, err = model.GetObjet(idx, item)
				if err != nil {
					return nil, err
				}

				if exists {
					addItem(item)
				}
			}
		}
	}

	// Items by cache
	cache := tx.getCache(model.From)
	for _, item := range cache {
		addItem(item)
	}

	next := true
	for _, item := range items {
		ok := Validate(item, s.conditions)
		if !ok {
			continue
		}
		next := addResult(item)
		if !next {
			break
		}
	}

	if onlyKeys {
		return result, nil
	}

	// Items by data
	asc := s.Order(catalog.INDEX)
	err := model.For(func(idx string, item et.Json) (bool, error) {
		next = Validate(item, s.conditions)
		return next, nil
	}, asc, s.offset, s.limit, s.workers)
	if err != nil {
		return nil, err
	}

	if !next {
		return result, nil
	}

	return result, nil
}

/**
* One
* @param tx *Tx
* @return et.Json, error
**/
func (s *Where) One(tx *Tx) (et.Json, error) {
	rows, err := s.Run(tx)
	if err != nil {
		return et.Json{}, err
	}

	if len(rows) == 0 {
		return et.Json{}, nil
	}

	return rows[0], nil
}

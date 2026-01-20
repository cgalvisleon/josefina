package rds

import (
	"encoding/json"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/josefina/pkg/msg"
)

/**
* Wheres
**/
type Wheres struct {
	owner      *Model              `json:"-"`
	selects    []string            `json:"-"`
	keys       map[string][]string `json:"-"`
	asc        map[string]bool     `json:"-"`
	offset     int                 `json:"-"`
	limit      int                 `json:"-"`
	conditions []*Condition        `json:"-"`
	workers    int                 `json:"-"`
}

/**
* newWhere
* @param owner *Model
* @return *Wheres
**/
func newWhere(owner *Model) *Wheres {
	workers := 1
	if owner != nil && len(owner.Indexes) > workers {
		workers = len(owner.Indexes)
	}
	return &Wheres{
		owner:      owner,
		selects:    make([]string, 0),
		keys:       make(map[string][]string, 0),
		asc:        make(map[string]bool, 0),
		offset:     0,
		limit:      0,
		conditions: make([]*Condition, 0),
		workers:    workers,
	}
}

/**
* setOwner
* @param owner *Model
* @return *Wheres
**/
func (s *Wheres) setOwner(owner *Model) *Wheres {
	if owner == nil {
		return s
	}

	s.owner = owner
	if len(owner.Indexes) > s.workers {
		s.workers = len(owner.Indexes)
	}
	return s
}

/**
* ToJson
* @return []et.Json
**/
func (s *Wheres) ToJson() []et.Json {
	result := []et.Json{}
	for _, condition := range s.conditions {
		result = append(result, condition.ToJson())
	}

	return result
}

/**
* ByJson
* @param jsons []et.Json
* @return void
**/
func (s *Wheres) ByJson(jsons []et.Json) {
	for _, where := range jsons {
		condition := ToCondition(where)
		if condition != nil {
			s.Add(condition)
		}
	}
}

/**
* Add
* @param condition *Condition
* @return *Wheres
**/
func (s *Wheres) Add(condition *Condition) *Wheres {
	if len(s.conditions) > 0 && condition.Connector == NaC {
		condition.Connector = And
	}

	s.conditions = append(s.conditions, condition)
	return s
}

/**
* And
* @param condition *Condition
* @return *Wheres
**/
func (s *Wheres) And(condition *Condition) *Wheres {
	condition.Connector = And
	return s.Add(condition)
}

/**
* Or
* @param condition *Condition
* @return *Wheres
**/
func (s *Wheres) Or(condition *Condition) *Wheres {
	condition.Connector = Or
	return s.Add(condition)
}

/**
* Selects
* @param fields ...string
* @return *Wheres
**/
func (s *Wheres) Selects(fields ...string) *Wheres {
	if len(fields) == 0 {
		return s
	}

	for _, field := range fields {
		s.selects = append(s.selects, field)
	}

	return s
}

/**
* Asc
* @param field string
* @return *Wheres
**/
func (s *Wheres) Asc(field string) *Wheres {
	s.asc[field] = true
	return s
}

/**
* Desc
* @param field string
* @return *Wheres
**/
func (s *Wheres) Desc(field string) *Wheres {
	s.asc[field] = false
	return s
}

/**
* Order
* @param field string
* @return bool
**/
func (s *Wheres) Order(field string) bool {
	result, exists := s.asc[field]
	if !exists {
		result = true
	}

	return result
}

/**
* Limit
* @param page int, rows int
* @return *Wheres
**/
func (s *Wheres) Limit(page int, rows int) *Wheres {
	offset := (page - 1) * rows
	s.limit = rows
	s.offset = offset
	return s
}

/**
* Rows
* @param tx *Tx
* @return []et.Json, error
**/
func (s *Wheres) Rows(tx *Tx) ([]et.Json, error) {
	result := []et.Json{}
	model := s.owner
	if model == nil {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	add := func(item et.Json) {
		if len(s.selects) > 0 {
			item = item.Select(s.selects)
		}
		result = append(result, item)
	}

	validateItem := func(item et.Json, conditions []*Condition) {
		var ok bool
		for i, con := range conditions {
			tmp := con.ApplyToData(item)
			if i == 0 {
				ok = tmp
			} else if con.Connector == And {
				ok = ok && tmp
			} else if con.Connector == Or {
				ok = ok || tmp
			}

			if !ok {
				break
			}
		}

		if ok {
			add(item)
		}
	}

	st, err := model.source()
	if err != nil {
		return nil, err
	}

	if len(s.conditions) == 0 {
		// Items by data
		asc := s.Order(INDEX)
		err = st.Iterate(func(id string, src []byte) (bool, error) {
			item := et.Json{}
			err := json.Unmarshal(src, &item)
			if err != nil {
				return false, err
			}

			add(item)
			return true, nil
		}, asc, s.offset, s.limit, s.workers)
		if err != nil {
			return nil, err
		}

		// Items by cache
		cache := tx.getRecors(model.Name)
		for _, record := range cache {
			item := record.Data

			add(item)
		}

		return result, nil
	}

	cnds := []*Condition{}
	for _, con := range s.conditions {
		value := con.Value
		switch v := value.(type) {
		case *Wheres:
			var err error
			con.Value, err = v.Rows(tx)
			if err != nil {
				return nil, err
			}
		case Wheres:
			var err error
			con.Value, err = v.Rows(tx)
			if err != nil {
				return nil, err
			}
		}

		field := con.Field
		index, ok := model.index(field)
		if !ok {
			cnds = append(cnds, con)
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
	for field, keys := range s.keys {
		for _, key := range keys {
			indexes := map[string]bool{}
			exists, err := model.getIndex(field, key, indexes)
			if err != nil {
				return nil, err
			}
			if !exists {
				continue
			}

			for idx := range indexes {
				item := et.Json{}
				exists, err = model.getObjet(idx, item)
				if err != nil {
					return nil, err
				}

				if exists {
					add(item)
				}
			}
		}
	}

	if len(cnds) == 0 {
		return result, nil
	}

	// Items by data
	asc := s.Order(INDEX)
	err = st.Iterate(func(id string, src []byte) (bool, error) {
		item := et.Json{}
		err := json.Unmarshal(src, &item)
		if err != nil {
			return false, err
		}

		validateItem(item, cnds)
		return true, nil
	}, asc, s.offset, s.limit, s.workers)
	if err != nil {
		return nil, err
	}

	// Items by cache
	cache := tx.getRecors(model.Name)
	for _, record := range cache {
		item := record.Data

		validateItem(item, cnds)
	}

	return result, nil
}

func Where(condition *Condition) *Wheres {
	return &Wheres{
		conditions: []*Condition{condition},
	}
}

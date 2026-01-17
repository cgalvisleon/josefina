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
	owner      *Model       `json:"-"`
	conditions []*Condition `json:"-"`
}

/**
* newWhere
* @param owner *Model
* @return *Wheres
**/
func newWhere(owner *Model) *Wheres {
	return &Wheres{
		owner:      owner,
		conditions: make([]*Condition, 0),
	}
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
* Rows
* @param page int, rows int
* @return et.Items, error
**/
func (s *Wheres) Rows(tx *Tx) ([]et.Json, error) {
	result := []et.Json{}
	model := s.owner
	if model == nil {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	add := func(item et.Json) {
		result = append(result, item)
	}

	cons := []*Condition{}
	for _, con := range s.conditions {
		field := con.Field
		index, ok := model.index(field)
		if ok {
			keys := index.Keys()
			keys = con.ApplyToIndex(keys)
			for _, key := range keys {
				item := et.Json{}
				exists, err := model.getJson(key, item)
				if err != nil {
					return nil, err
				}
				if exists {
					add(item)
				}
			}

			continue
		}

		value := con.Value
		switch v := value.(type) {
		case *Wheres:
			var err error
			con.Value, err = v.Rows()
			if err != nil {
				return nil, err
			}
		case Wheres:
			var err error
			con.Value, err = v.Rows()
			if err != nil {
				return nil, err
			}
		}

		cons = append(cons, con)
	}

	st, err := model.source()
	if err != nil {
		return nil, err
	}

	worker := len(model.Fields)
	st.Iterate(func(id string, src []byte) (bool, error) {
		item := et.Json{}
		err := json.Unmarshal(src, &item)
		if err != nil {
			return false, err
		}

		var ok bool
		for i, con := range cons {
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

		return true, nil
	}, worker)

	return result, nil
}

package rds

import (
	"encoding/json"

	"github.com/cgalvisleon/et/et"
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
* applyToData
* @param data et.Json
* @return bool
**/
func (s *Wheres) applyToData(data et.Json) bool {
	result := true
	for _, condition := range s.conditions {
		if condition.Connector == And {
			data = data.Json(condition.Field)
		}
	}

	return result
}

/**
* Rows
* @param page int, rows int
* @return et.Items, error
**/
func (s *Wheres) Rows() (et.Items, error) {
	result := et.Items{}
	st.Iterate(func(id string, data []byte) bool {
		result := et.Json{}
		err := json.Unmarshal(data, &result)
		if err != nil {
			panic(err)
		}

		// logs.Debug("iterate:", result.ToString())
		return true
	}, 2)

	return result, nil
}

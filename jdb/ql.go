package jdb

import (
	"encoding/json"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type TypeQuery string

const (
	TpSelect  TypeQuery = "select"
	TpData    TypeQuery = "data"
	TpExists  TypeQuery = "exists"
	TpCounted TypeQuery = "count"
)

type Ql struct {
	DB           *DB       `json:"-"`
	Type         TypeQuery `json:"type"`
	Froms        []*Froms  `json:"froms"`
	Wheres       *Wheres   `json:"wheres"`
	Selects      []*Field  `json:"select"`
	Hidden       []string  `json:"hidden"`
	Details      []*Field  `json:"details"`
	Rollups      []*Field  `json:"rollups"`
	Joins        []*Detail `json:"joins"`
	GroupsBy     []*Field  `json:"group_by"`
	Havings      *Wheres   `json:"having"`
	OrdersByAsc  []*Field  `json:"order_by_asc"`
	OrdersByDesc []*Field  `json:"order_by_desc"`
	Page         int       `json:"page"`
	Rows         int       `json:"rows"`
	MaxRows      int       `json:"max_rows"`
	IsDebug      bool      `json:"is_debug"`
	tx           *Tx       `json:"-"`
}

/**
* newQuery
* @param model *Model, as string, tp TypeQuery
* @return *Ql
**/
func newQuery(model *Model, as string, tp TypeQuery) *Ql {
	if model.SourceField != nil {
		tp = TpData
	}
	maxRows := envar.GetInt("MAX_ROWS", 100)
	result := &Ql{
		Type:         tp,
		DB:           model.DB,
		Froms:        []*Froms{newFrom(model, as)},
		Selects:      make([]*Field, 0),
		Hidden:       make([]string, 0),
		Details:      make([]*Field, 0),
		Rollups:      make([]*Field, 0),
		Joins:        make([]*Detail, 0),
		GroupsBy:     make([]*Field, 0),
		OrdersByAsc:  make([]*Field, 0),
		OrdersByDesc: make([]*Field, 0),
		Page:         0,
		Rows:         0,
		MaxRows:      maxRows,
	}
	result.Wheres = newWhere(result)
	result.Havings = newWhere(result)

	return result
}

/**
* Serialize
* @return []byte, error
**/
func (s *Ql) Serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Ql) ToJson() et.Json {
	bt, err := s.Serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* SetDebug
* @param isDebug bool
* @return *Ql
**/
func (s *Ql) SetDebug(isDebug bool) *Ql {
	s.IsDebug = isDebug
	return s
}

/**
* Debug
* @return *Ql
**/
func (s *Ql) Debug() *Ql {
	s.IsDebug = true
	return s
}

/**
* From
* @param model *Model, as string
* @return *Ql
**/
func (s *Ql) From(model *Model, as string) *Ql {
	s.Froms = append(s.Froms, newFrom(model, as))
	main := s.Froms[0]
	if main == nil {
		return s
	}

	detail, ok := main.Model.Details[model.Name]
	if !ok {
		return s
	}

	s.Joins = append(s.Joins, detail)

	return s
}

/**
* Join
* @param model *Model, as string, keys map[string]string
* @return *Ql
**/
func (s *Ql) Join(model *Model, as string, keys map[string]string) *Ql {
	join := newDetail(model, as, keys, []string{}, false, false)
	s.Joins = append(s.Joins, join)

	return s
}

/**
* SelectByColumns
* @return *Ql
**/
func (s *Ql) Select(fields ...string) *Ql {
	for _, field := range fields {
		fld := FindField(s.Froms, field)
		if fld != nil {
			switch fld.TypeField {
			case TpColumn:
				s.Selects = append(s.Selects, fld)
			case TpAtrib:
				s.Selects = append(s.Selects, fld)
			case TpDetail:
				s.Details = append(s.Details, fld)
			case TpRollup:
				s.Rollups = append(s.Rollups, fld)
			}
		}
	}
	return s
}

/**
* Where
* @param condition *Condition
* @return *Ql
**/
func (s *Ql) Where(condition *Condition) *Ql {
	s.Wheres.Add(condition)
	return s
}

/**
* AllTx
* @param tx *Tx
* @return et.Items, error
**/
func (s *Ql) AllTx(tx *Tx) (et.Items, error) {
	return s.DB.Query(s)
}

/**
* All
* @return et.Items, error
**/
func (s *Ql) All() (et.Items, error) {
	return s.AllTx(nil)
}

/**
* OneTx
* @param tx *Tx
* @return et.Item, error
**/
func (s *Ql) OneTx(tx *Tx) (et.Item, error) {
	result, err := s.AllTx(tx)
	if err != nil {
		return et.Item{}, err
	}

	return result.First(), nil
}

/**
* One
* @param tx *Tx
* @return et.Item, error
**/
func (s *Ql) One() (et.Item, error) {
	return s.OneTx(nil)
}

/**
* ItExistsTx
* @param tx *Tx
* @return bool, error
**/
func (s *Ql) ItExistsTx(tx *Tx) (bool, error) {
	s.Type = TpExists
	result, err := s.AllTx(tx)
	if err != nil {
		return false, err
	}

	exists := result.First().Bool("exists")
	return exists, nil
}

/**
* ItExists
* @return bool, error
**/
func (s *Ql) ItExists() (bool, error) {
	return s.ItExistsTx(nil)
}

/**
* CountTx
* @param tx *Tx
* @return int, error
**/
func (s *Ql) CountTx(tx *Tx) (int, error) {
	s.Type = TpCounted
	result, err := s.AllTx(tx)
	if err != nil {
		return 0, err
	}

	count := result.First().Int("count")
	return count, nil
}

/**
* Count
* @return int, error
**/
func (s *Ql) Count() (int, error) {
	return s.CountTx(nil)
}

/**
* From
* @param model *Model, as string
* @return *Ql
**/
func From(model *Model, as string) *Ql {
	return newQuery(model, as, TpSelect)
}

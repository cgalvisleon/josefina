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

type Orders struct {
	Field *Field `json:"field"`
	Asc   bool   `json:"asc"`
}

type Ql struct {
	DB       *DB                    `json:"-"`
	Type     TypeQuery              `json:"type"`
	Froms    []*Froms               `json:"froms"`
	Joins    []*Joins               `json:"joins"`
	Wheres   *Wheres                `json:"wheres"`
	Selects  []*Field               `json:"select"`
	Hidden   []*Field               `json:"hidden"`
	Details  map[string]*Field      `json:"details"`
	Rollups  map[string]*Field      `json:"rollups"`
	Calcs    map[string]DataContext `json:"calcs"`
	GroupsBy []*Field               `json:"group_by"`
	Havings  *Wheres                `json:"having"`
	OrdersBy []*Orders              `json:"order_by"`
	Page     int                    `json:"page"`
	Rows     int                    `json:"rows"`
	MaxRows  int                    `json:"max_rows"`
	IsDebug  bool                   `json:"is_debug"`
	tx       *Tx                    `json:"-"`
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
		Type:     tp,
		DB:       model.DB,
		Froms:    []*Froms{newFrom(model, as)},
		Joins:    make([]*Joins, 0),
		Selects:  make([]*Field, 0),
		Hidden:   make([]*Field, 0),
		Details:  make(map[string]*Field),
		Rollups:  make(map[string]*Field),
		Calcs:    make(map[string]DataContext),
		GroupsBy: make([]*Field, 0),
		OrdersBy: make([]*Orders, 0),
		Page:     0,
		Rows:     0,
		MaxRows:  maxRows,
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
* Join
* @param model *Model, as string, keys map[string]string
* @return *Ql
**/
func (s *Ql) join(tp TypeJoin, model *Model, as string, keys map[string]string) *Ql {
	from := newFrom(model, as)
	s.Froms = append(s.Froms, from)

	rKeys := make(map[string]string)
	for k, fk := range keys {
		field := FindField(s.Froms, k)
		if field != nil {
			k = field.AS()
		}

		field = FindField(s.Froms, fk)
		if field != nil {
			fk = field.AS()
		}

		rKeys[k] = fk
	}

	join := newJoins(tp, from, rKeys)
	s.Joins = append(s.Joins, join)

	return s
}

/**
* From
* @param model *Model, as string
* @return *Ql
**/
func (s *Ql) From(model *Model, as string) *Ql {
	if len(s.Froms) == 0 {
		return s
	}

	main := s.Froms[0].Model
	detail, ok := main.Relations[model.Name]
	if !ok {
		return s
	}

	keys := detail.Keys
	return s.Join(model, as, keys)
}

/**
* Join
* @param model *Model, as string, keys map[string]string
* @return *Ql
**/
func (s *Ql) Join(model *Model, as string, keys map[string]string) *Ql {
	return s.join(TpJoin, model, as, keys)
}

/**
* LeftJoin
* @param model *Model, as string, keys map[string]string
* @return *Ql
**/
func (s *Ql) LeftJoin(model *Model, as string, keys map[string]string) *Ql {
	return s.join(TpLeft, model, as, keys)
}

/**
* RightJoin
* @param model *Model, as string, keys map[string]string
* @return *Ql
**/
func (s *Ql) RightJoin(model *Model, as string, keys map[string]string) *Ql {
	return s.join(TpRight, model, as, keys)
}

/**
* FullJoin
* @param model *Model, as string, keys map[string]string
* @return *Ql
**/
func (s *Ql) FullJoin(model *Model, as string, keys map[string]string) *Ql {
	return s.join(TpFull, model, as, keys)
}

/**
* SelectByColumns
* @return *Ql
**/
func (s *Ql) Select(fields ...string) *Ql {
	for _, field := range fields {
		fld := FindField(s.Froms, field)
		if fld != nil {
			switch fld.TypeColumn {
			case TpColumn:
				s.Selects = append(s.Selects, fld)
			case TpAtrib:
				s.Selects = append(s.Selects, fld)
			case TpDetail:
				s.Details[fld.Name] = fld
			case TpRollup:
				s.Rollups[fld.Name] = fld
			case TpCalc:
				fn, ok := fld.Column.From.calcs[fld.Name]
				if !ok {
					continue
				}
				s.Calcs[fld.Name] = fn
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
* WhereByKeys
* @param keys et.Json
* @return *Ql
**/
func (s *Ql) WhereByKeys(keys et.Json) *Ql {
	for k, v := range keys {
		s.Wheres.Add(Eq(k, v))
	}
	return s
}

/**
* WhereByConditions
* @param conditions []*Condition
* @return *Ql
**/
func (s *Ql) WhereByConditions(conditions []*Condition) *Ql {
	for _, condition := range conditions {
		s.Wheres.Add(condition)
	}
	return s
}

/**
* WhereByJson
* @param jsons []et.Json
* @return *Ql
**/
func (s *Ql) WhereByJson(jsons []et.Json) *Ql {
	s.Wheres.ByJson(jsons)
	return s
}

/**
* And
* @param condition *Condition
* @return *Ql
**/
func (s *Ql) And(condition *Condition) *Ql {
	s.Wheres.Add(condition)
	return s
}

/**
* Or
* @param condition *Condition
* @return *Ql
**/
func (s *Ql) Or(condition *Condition) *Ql {
	s.Wheres.Add(condition)
	return s
}

/**
* GroupsBy
* @param fields ...string
* @return *Ql
**/
func (s *Ql) GroupBy(fields ...string) *Ql {
	for _, name := range fields {
		fld := FindField(s.Froms, name)
		if fld != nil {
			s.GroupsBy = append(s.GroupsBy, fld)
		}
	}
	return s
}

/**
* Having
* @param condition []*Condition
* @return *Ql
**/
func (s *Ql) Having(condition []*Condition) *Ql {
	for _, cnd := range condition {
		s.Havings.Add(cnd)
	}
	return s
}

/**
* HavingsByJson
* @param jsons []et.Json
* @return *Ql
**/
func (s *Ql) HavingsByJson(jsons []et.Json) *Ql {
	s.Havings.ByJson(jsons)
	return s
}

/**
* ordersBy
* @param asc bool, fields ...string
* @return *Ql
**/
func (s *Ql) ordersBy(asc bool, fields ...string) *Ql {
	for _, name := range fields {
		fld := FindField(s.Froms, name)
		if fld != nil {
			s.OrdersBy = append(s.OrdersBy, &Orders{Field: fld, Asc: asc})
		}
	}
	return s
}

/**
* OrderBy
* @param fields ...string
* @return *Ql
**/
func (s *Ql) OrderBy(fields ...string) *Ql {
	return s.ordersBy(true, fields...)
}

/**
* OrderByAsc
* @param fields ...string
* @return *Ql
**/
func (s *Ql) OrderByAsc(fields ...string) *Ql {
	return s.ordersBy(true, fields...)
}

/**
* OrderByDesc
* @param fields ...string
* @return *Ql
**/
func (s *Ql) OrderByDesc(fields ...string) *Ql {
	return s.ordersBy(false, fields...)
}

/**
* Hiddens
* @param fields ...string
* @return *Ql
**/
func (s *Ql) Hiddens(fields ...string) *Ql {
	for _, name := range fields {
		fld := FindField(s.Froms, name)
		if fld != nil {
			s.Hidden = append(s.Hidden, fld)
		}
	}
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
* LimitTx
* @param tx *Tx, page, rows int
* @return *Ql
**/
func (s *Ql) LimitTx(tx *Tx, page, rows int) (et.Items, error) {
	s.Page = page
	s.Rows = rows
	return s.AllTx(tx)
}

/**
* Limit
* @param page, rows int
* @return *Ql
**/
func (s *Ql) Limit(page, rows int) (et.Items, error) {
	return s.LimitTx(nil, page, rows)
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

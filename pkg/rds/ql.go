package rds

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Ql struct {
	From       *From    `json:"froms"`
	Selects    []string `json:"selects"`
	Hidden     []string `json:"hidden"`
	Wheres     *Wheres  `json:"wheres"`
	GroupsBy   []string `json:"groups_by"`
	Having     *Wheres  `json:"having"`
	OrdersAsc  []string `json:"orders_asc"`
	OrdersDesc []string `json:"orders_desc"`
	Page       int      `json:"page"`
	Rows       int      `json:"rows"`
	MaxRows    int      `json:"max_rows"`
	IsDebug    bool     `json:"is_debug"`
	db         *DB      `json:"-"`
	tx         *Tx      `json:"-"`
	result     et.Items `json:"-"`
}

/**
* newQl: Creates a new ql
* @param tx *Tx, model *Model, as string
* @return *Ql
**/
func newQl(tx *Tx, model *Model, as string) *Ql {
	maxRows := envar.GetInt("MAX_ROWS", 1000)
	from := model.From.clone()
	from.setAs(as)
	if tx == nil {
		tx = newTx(model.db)
	}
	return &Ql{
		Froms: []*From{
			from,
		},
		Selects:    make([]*Field, 0),
		Hidden:     make([]string, 0),
		Wheres:     make([]*Wheres, 0),
		GroupsBy:   make([]*Field, 0),
		Having:     make([]*Wheres, 0),
		OrdersAsc:  make([]*Field, 0),
		OrdersDesc: make([]*Field, 0),
		MaxRows:    maxRows,
		db:         model.db,
		tx:         tx,
	}
}

/**
* Select: Selects the model
* @return et.Items, error
**/
func (s *Ql) All() (et.Items, error) {
	return et.Items{}, nil
}

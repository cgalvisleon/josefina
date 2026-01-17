package rds

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Ql struct {
	From       *Model   `json:"froms"`
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
func newQl(tx *Tx, model *Model) *Ql {
	maxRows := envar.GetInt("MAX_ROWS", 1000)
	return &Ql{
		From:       model,
		Selects:    make([]string, 0),
		Hidden:     make([]string, 0),
		Wheres:     newWhere(model),
		GroupsBy:   make([]string, 0),
		Having:     newWhere(model),
		OrdersAsc:  make([]string, 0),
		OrdersDesc: make([]string, 0),
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

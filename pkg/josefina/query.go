package josefina

import "github.com/cgalvisleon/et/et"

type Ql struct {
	Froms      []*From   `json:"froms"`
	Selects    []*Field  `json:"selects"`
	Hidden     []string  `json:"hidden"`
	Wheres     []*Wheres `json:"wheres"`
	GroupsBy   []*Field  `json:"groups_by"`
	Having     []*Wheres `json:"having"`
	OrdersAsc  []*Field  `json:"orders_asc"`
	OrdersDesc []*Field  `json:"orders_desc"`
	Page       int       `json:"page"`
	Rows       int       `json:"rows"`
	MaxRows    int       `json:"max_rows"`
	IsDebug    bool      `json:"is_debug"`
	db         *DB       `json:"-"`
	tx         *Tx       `json:"-"`
	result     et.Items  `json:"-"`
}

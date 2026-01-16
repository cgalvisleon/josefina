package rds

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

type Tx struct {
	StartedAt time.Time            `json:"started_at"`
	EndedAt   time.Time            `json:"ended_at"`
	Id        string               `json:"id"`
	data      map[string][]et.Json `json:"-"`
}

func newTx() *Tx {
	return &Tx{
		StartedAt: time.Now(),
		Id:        reg.GenULID("transaction"),
		data:      make(map[string][]et.Json, 0),
	}
}

/**
* Add: Adds data to the transaction
* @param name string, data et.Json
**/
func (s *Tx) Add(name string, data et.Json) {
	s.data[name] = append(s.data[name], data)
}

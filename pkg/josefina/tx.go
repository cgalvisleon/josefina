package josefina

import (
	"time"

	"github.com/cgalvisleon/et/reg"
)

type Tx struct {
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Id        string    `json:"id"`
	db        *DB       `json:"-"`
}

type TxError struct {
	Database string `json:"database"`
	Id       int    `json:"id"`
	Error    []byte `json:"error"`
}

func newTx(db *DB) *Tx {
	return &Tx{
		StartedAt: time.Now(),
		Id:        reg.GenULID("transaction"),
		db:        db,
	}
}

package jdb

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

type TransactionStatus string

const (
	PENDING     TransactionStatus = "pending"
	ROLLED_BACK TransactionStatus = "rolled_back"
	COMMITTED   TransactionStatus = "committed"
)

/**
* Transaction: Records a single DML operation within a Tx, including old and new state for rollback.
**/
type Transaction struct {
	Database  string            `json:"database"`
	Schema    string            `json:"schema"`
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Command   Cmd               `json:"command"`
	Idx       string            `json:"id"`
	Data      et.Json           `json:"new"`
	Status    TransactionStatus `json:"status"`
	model     *Model            `json:"-"`
}

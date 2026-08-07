package jdb

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
)

type Error struct {
	CreatedAt     time.Time `json:"created_at"`
	ID            string    `json:"id"`
	TransactionId string    `json:"transaction_id"`
	Code          string    `json:"code"`
	Message       string    `json:"message"`
	Data          et.Json   `json:"data"`
}

func (s *Error) ToJson() et.Json {
	return et.Json{
		"created_at":     s.CreatedAt,
		"id":             s.ID,
		"transaction_id": s.TransactionId,
		"code":           s.Code,
		"message":        s.Message,
		"data":           s.Data,
	}
}

type Errors struct {
	errors map[string]*Error
	mu     sync.RWMutex
	store  *Model
}

/**
* loadErrors: Initializes the internal errors model for a database.
* @param db *DB
* @return error
**/
func (s *DB) loadErrors() (*Errors, error) {
	store, err := s.loadModel(sysSchema, "errors", 1, true)
	if err != nil {
		return nil, err
	}

	return &Errors{
		errors: make(map[string]*Error),
		store:  store,
	}, nil
}

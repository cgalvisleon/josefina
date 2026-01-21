package rds

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Session struct {
	CreatedAt time.Time `json:"created_at"`
	Id        string    `json:"id"`
	Username  string    `json:"username"`
}

/**
* NewSession: Creates a new session
* @param username string
* @return *Session
**/
func NewSession(username string) *Session {
	id := reg.GenULID("session")
	return &Session{
		CreatedAt: time.Now(),
		Id:        id,
		Username:  username,
	}
}

/**
* getTx: Returns the transaction for the session
* @param tx *Tx
* @return (*Tx, bool)
**/
func (s *Session) getTx(tx *Tx) (*Tx, bool) {
	if tx != nil {
		return tx, false
	}

	id := reg.GenULID("transaction")
	tx = &Tx{
		StartedAt:    timezone.Now(),
		EndedAt:      time.Time{},
		Session:      s.Id,
		Id:           id,
		Transactions: make([]*transaction, 0),
	}
	return tx, true
}

func (s *Session) Insert(model *Model, new et.Json) (et.Json, error) {
	return model.Insert(nil, new)
}

var sessions []*Session

func init() {
	sessions = make([]*Session, 0)
}

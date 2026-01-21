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

func (s *Session) Insert(tx *Tx, model *Model, new et.Json) (et.Json, error) {
	tx, commit := s.getTx(tx)
	result, err := model.insert(tx, new)
	if err != nil {
		return nil, err
	}

	
	return result, nil
}

var sessions []*Session

func init() {
	sessions = make([]*Session, 0)
}

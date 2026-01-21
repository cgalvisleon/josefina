package rds

import (
	"time"

	"github.com/cgalvisleon/et/reg"
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

var sessions []*Session

func init() {
	sessions = make([]*Session, 0)
}

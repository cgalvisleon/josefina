package server

import (
	"time"

	"github.com/cgalvisleon/et/reg"
)

type TpConnection int

const (
	HTTP TpConnection = iota
	WebSocket
	TCP
)

type Status int

const (
	Connected Status = iota
	Disconnected
)

type Session struct {
	CreatedAt time.Time    `json:"created_at"`
	ID        string       `json:"id"`
	Username  string       `json:"username"`
	Address   string       `json:"address"`
	Status    Status       `json:"status"`
	Type      TpConnection `json:"type"`
	Database  string       `json:"database"`
}

/**
* NewSession
* @param username, address string, tp TpConnection, database string
* @return *Session
**/
func NewSession(username, address string, tp TpConnection, database string) *Session {
	return &Session{
		CreatedAt: time.Now(),
		ID:        reg.ULID(),
		Username:  username,
		Address:   address,
		Status:    Connected,
		Type:      tp,
		Database:  database,
	}
}

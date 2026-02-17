package node

import (
	"time"

	"github.com/cgalvisleon/et/reg"
)

type Status string

const (
	Active       Status = "active"
	Archived     Status = "archived"
	Canceled     Status = "canceled"
	OfSystem     Status = "of_system"
	ForDelete    Status = "for_delete"
	Pending      Status = "pending"
	Approved     Status = "approved"
	Rejected     Status = "rejected"
	Failed       Status = "failed"
	Processed    Status = "processed"
	Connected    Status = "connected"
	Disconnected Status = "disconnected"
)

type TpConnection int

const (
	HTTP TpConnection = iota
	WebSocket
	TCP
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

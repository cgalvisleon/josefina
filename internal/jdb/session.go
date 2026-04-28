package jdb

import "time"

type Status string

const ()

type TpConnection string

const (
	HTTP      TpConnection = "http"
	WebSocket TpConnection = "websocket"
	TCP       TpConnection = "tcp"
)

type Session struct {
	CreatedAt time.Time    `json:"created_at"`
	ID        string       `json:"id"`
	Username  string       `json:"username"`
	Address   string       `json:"address"`
	Status    Status       `json:"status"`
	Type      TpConnection `json:"type"`
	Device    string       `json:"device"`
	Database  string       `json:"database"`
	Token     string       `json:"-"`
}

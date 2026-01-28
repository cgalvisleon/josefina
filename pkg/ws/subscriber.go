package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Subscriber struct {
	Created_at time.Time           `json:"created_at"`
	Id         string              `json:"id"`
	Name       string              `json:"name"`
	Addr       string              `json:"addr"`
	Channels   map[string]*Channel `json:"channels"`
	socket     *websocket.Conn     `json:"-"`
	outbound   chan []byte         `json:"-"`
	mutex      sync.RWMutex        `json:"-"`
}

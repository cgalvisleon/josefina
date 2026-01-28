package ws

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/timezone"
	"github.com/gorilla/websocket"
)

type Subscriber struct {
	Created_at time.Time           `json:"created_at"`
	Name       string              `json:"name"`
	Addr       string              `json:"addr"`
	Channels   map[string]*Channel `json:"channels"`
	socket     *websocket.Conn     `json:"-"`
	outbound   chan []byte         `json:"-"`
	mutex      sync.RWMutex        `json:"-"`
}

/**
* newSubscriber
* @param name string, socket *websocket.Conn
* @return *Subscriber
**/
func newSubscriber(name string, socket *websocket.Conn) *Subscriber {
	return &Subscriber{
		Created_at: timezone.Now(),
		Name:       name,
		Addr:       socket.RemoteAddr().String(),
		Channels:   make(map[string]*Channel),
		socket:     socket,
		outbound:   make(chan []byte),
		mutex:      sync.RWMutex{},
	}
}

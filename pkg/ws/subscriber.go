package ws

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
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
	hub        *Hub                `json:"-"`
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

/**
* read
**/
func (s *Subscriber) read() {
	defer func() {

	}()

	for {
		msgType, message, err := s.socket.ReadMessage()
		if err != nil {
			break
		}

		s.listener(msgType, message)
	}
}

/**
* write
**/
func (s *Subscriber) write() {
	for message := range s.outbound {
		s.socket.WriteMessage(websocket.BinaryMessage, message)
	}

	s.socket.WriteMessage(websocket.CloseMessage, []byte{})
}

/**
* listener
* @param message []byte
**/
func (s *Subscriber) listener(messageType int, message []byte) {
	msg, err := DecodeMessage(message)
	if err != nil {
		logs.Error(err)
		return
	}

	logs.Info(msg.ToString())
}

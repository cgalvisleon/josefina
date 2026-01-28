package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/gorilla/websocket"
)

const (
	TextMessage   int = 1
	BinaryMessage int = 2
	CloseMessage  int = 8
	PingMessage   int = 9
	PongMessage   int = 10
)

type Outbound struct {
	messageType int
	message     []byte
}

type Subscriber struct {
	Created_at time.Time           `json:"created_at"`
	Name       string              `json:"name"`
	Addr       string              `json:"addr"`
	Channels   map[string]*Channel `json:"channels"`
	socket     *websocket.Conn     `json:"-"`
	outbound   chan Outbound       `json:"-"`
	mutex      sync.RWMutex        `json:"-"`
	hub        *Hub                `json:"-"`
	ctx        context.Context     `json:"-"`
}

/**
* newSubscriber
* @param name string, socket *websocket.Conn
* @return *Subscriber
**/
func newSubscriber(hub *Hub, ctx context.Context, username string, socket *websocket.Conn) *Subscriber {
	return &Subscriber{
		Created_at: timezone.Now(),
		Name:       username,
		Addr:       socket.RemoteAddr().String(),
		Channels:   make(map[string]*Channel),
		socket:     socket,
		outbound:   make(chan Outbound),
		mutex:      sync.RWMutex{},
		hub:        hub,
		ctx:        ctx,
	}
}

/**
* read
**/
func (s *Subscriber) read() {
	for {
		msgType, message, err := s.socket.ReadMessage()
		if err != nil {
			s.hub.unregister <- s
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
		s.socket.WriteMessage(message.messageType, message.message)
	}

	s.socket.WriteMessage(websocket.CloseMessage, []byte{})
}

/**
* listener
* @param message []byte
**/
func (s *Subscriber) listener(messageType int, message []byte) {
	msg, err := DecodeMessage(messageType, message)
	if err != nil {
		s.error(err)
		return
	}

	logs.Info(msg.ToString())
}

/**
* send
* @param msg Message
**/
func (s *Subscriber) send(tp int, bt []byte) {
	s.outbound <- Outbound{
		messageType: tp,
		message:     bt,
	}
}

/**
* error
* @param err error
**/
func (s *Subscriber) error(err error) {
	msg := et.Item{
		Ok: false,
		Result: et.Json{
			"message": err.Error(),
		},
	}
	bt, err := json.Marshal(msg)
	if err != nil {
		return
	}

	s.send(TextMessage, bt)
}

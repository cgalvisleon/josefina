package ws

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/gorilla/websocket"
)

type Status string

const (
	Pending      Status = "pending"
	Connected    Status = "connected"
	Disconnected Status = "disconnected"
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
	Created_at time.Time       `json:"created_at"`
	Name       string          `json:"name"`
	Addr       string          `json:"addr"`
	Status     Status          `json:"status"`
	Channels   []string        `json:"channels"`
	socket     *websocket.Conn `json:"-"`
	outbound   chan Outbound   `json:"-"`
	mutex      sync.RWMutex    `json:"-"`
	hub        *Hub            `json:"-"`
	ctx        context.Context `json:"-"`
}

/**
* newSubscriber
* @param name string, socket *websocket.Conn
* @return *Subscriber
**/
func newSubscriber(hub *Hub, ctx context.Context, username string, socket *websocket.Conn) *Subscriber {
	return &Subscriber{
		Created_at: timezone.Now(),
		Status:     Pending,
		Name:       username,
		Addr:       socket.RemoteAddr().String(),
		Channels:   []string{},
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
		_, message, err := s.socket.ReadMessage()
		if err != nil {
			s.hub.unregister <- s
			break
		}

		s.listener(message)
	}
}

/**
* write
**/
func (s *Subscriber) write() {
	for message := range s.outbound {
		s.socket.WriteMessage(TextMessage, message.message)
	}

	s.socket.WriteMessage(CloseMessage, []byte{})
}

/**
* listener
* @param message []byte
**/
func (s *Subscriber) listener(message []byte) {
	ms, err := DecodeMessage(message)
	if err != nil {
		s.error(err)
		return
	}

	if ms.Channel != "" {
		s.hub.Publish(ms.Channel, ms)
	} else if len(ms.To) > 0 {
		s.hub.SendTo(ms.To, ms)
	}

	logs.Info(ms.ToString())
}

/**
* send
* @param tp int, bt []byte
**/
func (s *Subscriber) send(tp int, bt []byte) {
	s.outbound <- Outbound{
		messageType: tp,
		message:     bt,
	}
}

/**
* SendText
* @param message string
**/
func (s *Subscriber) sendText(message string) {
	s.send(TextMessage, []byte(message))
}

/**
* SendObject
* @param message et.Json
**/
func (s *Subscriber) sendObject(message et.Json) {
	bt, err := json.Marshal(message)
	if err != nil {
		return
	}
	s.send(BinaryMessage, bt)
}

/**
* error
* @param err error
**/
func (s *Subscriber) error(err error) {
	ms := et.Item{
		Ok: false,
		Result: et.Json{
			"message": err.Error(),
		},
	}
	bt, err := json.Marshal(ms)
	if err != nil {
		return
	}

	s.send(TextMessage, bt)
}

/**
* sendHola
**/
func (s *Subscriber) sendHola() {
	ms := et.Item{
		Ok: true,
		Result: et.Json{
			"message": msg.MSG_HOLA,
		},
	}
	bt, err := json.Marshal(ms)
	if err != nil {
		return
	}

	s.send(TextMessage, bt)
}

/**
* addChannel
* @param channel string
**/
func (s *Subscriber) addChannel(channel string) {
	idx := slices.IndexFunc(s.Channels, func(c string) bool {
		return c == channel
	})
	if idx != -1 {
		return
	}
	s.Channels = append(s.Channels, channel)
}

/**
* removeChannel
* @param channel string
**/
func (s *Subscriber) removeChannel(channel string) {
	idx := slices.IndexFunc(s.Channels, func(c string) bool {
		return c == channel
	})
	if idx == -1 {
		return
	}

	s.Channels = append(s.Channels[:idx], s.Channels[idx+1:]...)
}

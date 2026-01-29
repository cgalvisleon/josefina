package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/gorilla/websocket"
)

const (
	packageName = "WebSocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	Channels        map[string]*Channel      `json:"channels"`
	Subscribers     map[string]*Subscriber   `json:"subscribers"`
	register        chan *Subscriber         `json:"-"`
	unregister      chan *Subscriber         `json:"-"`
	onConnection    []func(*Subscriber)      `json:"-"`
	onDisconnection []func(*Subscriber)      `json:"-"`
	onPublish       map[string]func(Message) `json:"-"`
	onSubscribe     map[string]func(Message) `json:"-"`
	onStack         map[string]func(Message) `json:"-"`
	mu              *sync.RWMutex            `json:"-"`
	isStart         bool                     `json:"-"`
}

/**
* NewWs
* @return *Hub
**/
func NewWs() *Hub {
	result := &Hub{
		Channels:        make(map[string]*Channel),
		Subscribers:     make(map[string]*Subscriber),
		register:        make(chan *Subscriber),
		unregister:      make(chan *Subscriber),
		onConnection:    make([]func(*Subscriber), 0),
		onDisconnection: make([]func(*Subscriber), 0),
		onPublish:       make(map[string]func(Message)),
		onSubscribe:     make(map[string]func(Message)),
		onStack:         make(map[string]func(Message)),
		mu:              &sync.RWMutex{},
		isStart:         false,
	}
	return result
}

/**
* Upgrader
* @params w http.ResponseWriter, r *http.Request
* @return  *websocket.Conn, error
**/
func Upgrader(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

/**
* run
*
 */
func (s *Hub) run() {
	for {
		select {
		case client := <-s.register:
			s.onConnect(client)
		case client := <-s.unregister:
			s.onDisconnect(client)
		}
	}
}

/**
* Start
**/
func (s *Hub) Start() {
	if s.isStart {
		return
	}

	s.isStart = true
	go s.run()
}

/**
* defOnConnect
* @param *Subscriber client
**/
func (s *Hub) onConnect(client *Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Subscribers[client.Name] = client
	logs.Logf(packageName, "Client connected: %s", client.Name)
	for _, fn := range s.onConnection {
		fn(client)
	}
}

/**
* onDisconnect
* @param *Subscriber client
**/
func (s *Hub) onDisconnect(client *Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.Subscribers[client.Name]
	if ok {
		s.Subscribers[client.Name].Status = Disconnected
		for _, fn := range s.onDisconnection {
			fn(client)
		}

		delete(s.Subscribers, client.Name)
	}
}

/**
* Connect
* @param socket *websocket.Conn, context.Context
* @return *Subscriber, error
**/
func (s *Hub) Connect(socket *websocket.Conn, ctx context.Context) (*Subscriber, error) {
	username := ctx.Value("username").(string)
	if !utility.ValidStr(username, 0, []string{""}) {
		return nil, fmt.Errorf(msg.MSG_ARG_REQUIRED, "username")
	}

	client, ok := s.Subscribers[username]
	if ok {
		client.Addr = socket.RemoteAddr().String()
		client.socket = socket
		return client, nil
	}

	client = newSubscriber(s, ctx, username, socket)
	s.register <- client

	go client.write()
	go client.read()

	return client, nil
}

/**
* OnConnection
* @param fn func(*Subscriber)
**/
func (s *Hub) OnConnection(fn func(*Subscriber)) {
	s.onConnection = append(s.onConnection, fn)
}

/**
* OnDisconnection
* @param fn func(*Subscriber)
**/
func (s *Hub) OnDisconnection(fn func(*Subscriber)) {
	s.onDisconnection = append(s.onDisconnection, fn)
}

/**
* OnPublish
* @param channel string, fn func(Message)
**/
func (s *Hub) OnPublish(channel string, fn func(Message)) {
	s.onPublish[channel] = fn
}

/**
* OnSubscribe
* @param channel string, fn func(Message)
**/
func (s *Hub) OnSubscribe(channel string, fn func(Message)) {
	s.onSubscribe[channel] = fn
}

/**
* OnStack
* @param channel string, fn func(Message)
**/
func (s *Hub) OnStack(channel string, fn func(Message)) {
	s.onStack[channel] = fn
}

/**
* SendTo
* @param to []string, message Message
**/
func (s *Hub) SendTo(to []string, message Message) {
	for _, username := range to {
		client, ok := s.Subscribers[username]
		if ok {
			if len(message.Data) > 0 {
				client.sendObject(message.Data)
			} else if len(message.Message) > 0 {
				client.sendText(message.Message)
			}
		}
	}
}

/**
* Topic
* @param channel string
* @return *Channel
**/
func (s *Hub) Topic(channel string) *Channel {
	ch := newChannel(channel, TpTopic)
	s.mu.Lock()
	s.Channels[channel] = ch
	s.mu.Unlock()
	return ch
}

/**
* Queue
* @param channel string
* @return *Channel
**/
func (s *Hub) Queue(channel string) *Channel {
	ch := newChannel(channel, TpQueue)
	s.mu.Lock()
	s.Channels[channel] = ch
	s.mu.Unlock()
	return ch
}

/**
* Stack
* @param channel string
* @return *Channel
**/
func (s *Hub) Stack(channel string) *Channel {
	ch := newChannel(channel, TpStack)
	s.mu.Lock()
	s.Channels[channel] = ch
	s.mu.Unlock()
	return ch
}

/**
* Subscribe
* @param cache string, subscribe string
* @return error
**/
func (s *Hub) Subscribe(cache string, subscribe string) error {
	ch, ok := s.Channels[cache]
	if !ok {
		return fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, cache)
	}

	client, ok := s.Subscribers[subscribe]
	if !ok {
		return fmt.Errorf(msg.MSG_USER_NOT_FOUND, subscribe)
	}

	ch.subscriber(client.Name)
	client.addChannel(ch.Name)
	return nil
}

/**
* Unsubscribe
* @param cache string, subscribe string
* @return error
**/
func (s *Hub) Unsubscribe(cache string, subscribe string) error {
	ch, ok := s.Channels[cache]
	if !ok {
		return fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, cache)
	}

	client, ok := s.Subscribers[subscribe]
	if !ok {
		return fmt.Errorf(msg.MSG_USER_NOT_FOUND, subscribe)
	}

	ch.remove(client.Name)
	client.removeChannel(ch.Name)
	return nil
}

/**
* Publish
* @param channel string, message Message
**/
func (s *Hub) Publish(channel string, message Message) error {
	s.mu.RLock()
	ch, ok := s.Channels[channel]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, channel)
	}

	switch ch.Type {
	case TpQueue:
		// TODO: Implement queue logic
	case TpTopic:
		// TODO: Implement topic logic
		s.SendTo(ch.Subscribers, message)
	}

	return nil
}

package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/gorilla/websocket"
)

const (
	packageName = "WebSocket"
	version     = "0.0.1"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	port            int
	channels        map[string]*Channel
	subscribers     map[string]*Subscriber
	register        chan *Subscriber
	unregister      chan *Subscriber
	onConnection    []func(*Subscriber)
	onDisconnection []func(*Subscriber)
	onPublish       map[string]func(Message)
	onSubscribe     map[string]func(Message)
	onStack         map[string]func(Message)
	mu              *sync.RWMutex
	isStart         bool
}

/**
* NewWs
* @return *Hub
**/
func NewWs() *Hub {
	port := envar.GetInt("WS_PORT", 3030)
	result := &Hub{
		port:            port,
		channels:        make(map[string]*Channel),
		subscribers:     make(map[string]*Subscriber),
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

	s.subscribers[client.Name] = client
	logs.Logf(packageName, "Connected: %s", client.Name)
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

	_, ok := s.subscribers[client.Name]
	if ok {
		s.subscribers[client.Name].Status = Disconnected
		for _, fn := range s.onDisconnection {
			fn(client)
		}

		delete(s.subscribers, client.Name)
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

	client, ok := s.subscribers[username]
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
		client, ok := s.subscribers[username]
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
* Subscribe
* @param action TypeAction, cache string, subscribe string
* @return error
**/
func (s *Hub) Subscribe(action TypeAction, cache string, subscribe string) error {
	return nil
}

/**
* Publish
* @param channel string, message Message
**/
func (s *Hub) Publish(channel string, message Message) error {
	s.mu.RLock()
	ch, ok := s.channels[channel]
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

package ws

import (
	"context"
	"fmt"
	"net/http"
	"slices"
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
	Channels        map[string]*Channel     `json:"channels"`
	Subscribers     map[string]*Subscriber  `json:"subscribers"`
	register        chan *Subscriber        `json:"-"`
	unregister      chan *Subscriber        `json:"-"`
	onConnection    []func(*Subscriber)     `json:"-"`
	onDisconnection []func(*Subscriber)     `json:"-"`
	onChannel       []func(Channel)         `json:"-"`
	onRemove        []func(string)          `json:"-"`
	onSend          []func(string, Message) `json:"-"`
	mu              *sync.RWMutex           `json:"-"`
	isStart         bool                    `json:"-"`
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
		onChannel:       make([]func(Channel), 0),
		onRemove:        make([]func(string), 0),
		onSend:          make([]func(string, Message), 0),
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
* OnChannel
* @param fn func(Channel)
**/
func (s *Hub) OnChannel(fn func(Channel)) {
	s.onChannel = append(s.onChannel, fn)
}

/**
* OnRemove
* @param fn func(string)
**/
func (s *Hub) OnRemove(fn func(string)) {
	s.onRemove = append(s.onRemove, fn)
}

/**
* OnSend
* @param fn func(string, Message)
 */
func (s *Hub) OnSend(fn func(string, Message)) {
	s.onSend = append(s.onSend, fn)
}

/**
* addChannel
* @param *Channel ch
**/
func (s *Hub) addChannel(ch *Channel) {
	s.mu.Lock()
	s.Channels[ch.Name] = ch
	s.mu.Unlock()
	for _, fn := range s.onChannel {
		fn(*ch)
	}
}

/**
* Topic
* @param channel string
* @return *Channel
*
 */
func (s *Hub) Topic(channel string) *Channel {
	ch := newChannel(channel, TpTopic)
	s.addChannel(ch)
	return ch
}

/**
* Queue
* @param channel string
* @return *Channel
**/
func (s *Hub) Queue(channel string) *Channel {
	ch := newChannel(channel, TpQueue)
	s.addChannel(ch)
	return ch
}

/**
* Stack
* @param channel string
* @return *Channel
**/
func (s *Hub) Stack(channel string) *Channel {
	ch := newChannel(channel, TpStack)
	s.addChannel(ch)
	return ch
}

/**
* Remove
* @param channel string
* @return error
**/
func (s *Hub) Remove(channel string) error {
	s.mu.Lock()
	ch, ok := s.Channels[channel]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf(msg.MSG_CHANNEL_NOT_FOUND, channel)
	}
	s.mu.Unlock()

	for _, subscribe := range ch.Subscribers {
		client, ok := s.Subscribers[subscribe]
		if ok {
			client.removeChannel(channel)
		}
	}

	s.mu.Lock()
	delete(s.Channels, channel)
	s.mu.Unlock()
	for _, fn := range s.onRemove {
		fn(channel)
	}
	return nil
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
* SendTo
* @param to []string, message Message
**/
func (s *Hub) SendTo(to []string, message Message) {
	for _, username := range to {
		client, ok := s.Subscribers[username]
		if ok {
			idx := slices.IndexFunc(message.Ignored, func(user string) bool {
				return user == username
			})
			if idx != -1 {
				continue
			}

			if len(message.Data) > 0 {
				client.sendObject(message.Data)
				for _, fn := range s.onSend {
					fn(username, message)
				}
			} else if len(message.Message) > 0 {
				client.sendText(message.Message)
				for _, fn := range s.onSend {
					fn(username, message)
				}
			}
		}
	}
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
	case TpStack:
	case TpTopic:
		s.SendTo(ch.Subscribers, message)
	}

	return nil
}

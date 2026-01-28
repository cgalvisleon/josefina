package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/utility"
	"github.com/cgalvisleon/josefina/pkg/msg"
	"github.com/gorilla/websocket"
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
	mutex           *sync.RWMutex
	onConnection    []func(*Subscriber)
	onDisconnection []func(*Subscriber)
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
		mutex:           &sync.RWMutex{},
		onConnection:    make([]func(*Subscriber), 0),
		onDisconnection: make([]func(*Subscriber), 0),
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
			s.onUnregister(client)
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
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.subscribers[client.Name] = client
	for _, fn := range s.onConnection {
		fn(client)
	}
}

/**
* defOnDisconnect
* @param *Subscriber client
**/
func (s *Hub) onDisconnect(client *Subscriber) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.subscribers, client.Name)
	for _, fn := range s.onDisconnection {
		fn(client)
	}
}

/**
* defOnDisconnect
* @param *Subscriber client
**/
func (s *Hub) onUnregister(client *Subscriber) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, ok := s.subscribers[client.Name]
	if ok {
		s.subscribers[client.Name].Status = Disconnected
	}
}

/**
* connect
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

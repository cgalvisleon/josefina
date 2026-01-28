package ws

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	host            string
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
	result := &Hub{
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
* Start
**/
func (s *Hub) Start() {
	s.isStart = true
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
* connect
* @param socket *websocket.Conn, username string
* @return *Subscriber, error
**/
func (s *Hub) connect(socket *websocket.Conn, username string) (*Subscriber, error) {
	client, ok := s.subscribers[username]
	if ok {
		client.Addr = socket.RemoteAddr().String()
		client.socket = socket
		return client, nil
	}

	client = newSubscriber(s, username, socket)
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

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

type Ws struct {
	host        string
	channels    map[string]*Channel
	subscribers map[string]*Subscriber
	register    chan *Subscriber
	unregister  chan *Subscriber
	mutex       *sync.RWMutex
	isStart     bool
}

/**
* NewWs
* @return *Ws
**/
func NewWs() *Ws {
	return &Ws{
		channels:    make(map[string]*Channel),
		subscribers: make(map[string]*Subscriber),
		register:    make(chan *Subscriber),
		unregister:  make(chan *Subscriber),
		mutex:       &sync.RWMutex{},
		isStart:     false,
	}
}

/**
* start
**/
func (s *Ws) start() {
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
* onConnect
* @param *Subscriber client
**/
func (s *Ws) onConnect(client *Subscriber) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.subscribers[client.Id] = client
}

/**
* onDisconnect
* @param *Subscriber client
**/
func (s *Ws) onDisconnect(client *Subscriber) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.subscribers, client.Id)
}

func (s *Ws) connect(socket *websocket.Conn, clientId, name string) (*Subscriber, error) {
	client := s.getClient(clientId)
	if client != nil {
		return client, nil
	}

	client, isNew := newSubscriber(s, socket, clientId, name)
	if isNew {
		h.register <- client

		go client.write()
		go client.read()
	}

	return client, nil
}

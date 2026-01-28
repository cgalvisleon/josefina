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
	host       string
	clients    []*Subscriber
	channels   []*Channel
	register   chan *Subscriber
	unregister chan *Subscriber
	mutex      *sync.RWMutex
	isStart    bool
}

/**
* newHub
* @return *Hub
**/
func newHub() *Hub {
	return &Hub{
		clients:    []*Subscriber{},
		channels:   []*Channel{},
		register:   make(chan *Subscriber),
		unregister: make(chan *Subscriber),
		mutex:      &sync.RWMutex{},
		isStart:    false,
	}
}

/**
* start
**/
func (h *Hub) start() {
	h.isStart = true
	for {
		select {
		case client := <-h.register:
			h.onConnect(client)
		case client := <-h.unregister:
			h.onDisconnect(client)
		}
	}
}

/**
* onConnect
* @param *Subscriber client
**/
func (h *Hub) onConnect(client *Subscriber) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients = append(h.clients, client)
}

/**
* onDisconnect
* @param *Subscriber client
**/
func (h *Hub) onDisconnect(client *Subscriber) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for i, c := range h.clients {
		if c == client {
			h.clients = append(h.clients[:i], h.clients[i+1:]...)
			break
		}
	}
}

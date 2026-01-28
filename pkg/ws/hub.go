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
	isInit     bool
}

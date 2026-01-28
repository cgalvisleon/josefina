package ws

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

type Message struct {
	Created_at time.Time   `json:"created_at"`
	Id         string      `json:"id"`
	From       et.Json     `json:"from"`
	Channel    string      `json:"channel"`
	To         string      `json:"to"`
	Ignored    []string    `json:"-"`
	Data       interface{} `json:"data"`
}

/**
* newMessage
* @param from et.Json, to string, data interface{}
* @return *Message
**/
func newMessage(from et.Json, to string, data interface{}) *Message {
	id := reg.UUID()
	return &Message{
		Created_at: time.Now(),
		Id:         id,
		From:       from,
		Channel:    "",
		To:         to,
		Ignored:    []string{},
		Data:       data,
	}
}

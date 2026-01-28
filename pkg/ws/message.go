package ws

import (
	"encoding/json"
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
* ToJson
* @return et.Json
**/
func (s *Message) ToJson() et.Json {
	bt, err := json.Marshal(s)
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* ToString
* @return string
**/
func (s *Message) ToString() string {
	return s.ToJson().ToString()
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

/**
* DecodeMessage
* @param []byte
* @return Message
**/
func DecodeMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	if err != nil {
		return Message{}, err
	}

	return m, nil
}

package ws

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
)

type Message struct {
	Created_at time.Time `json:"created_at"`
	Id         string    `json:"id"`
	From       et.Json   `json:"from"`
	Channel    string    `json:"channel"`
	To         []string  `json:"to"`
	Ignored    []string  `json:"-"`
	Data       et.Json   `json:"data"`
	Message    string    `json:"message"`
}

/**
* Bytes
* @return ([]byte, error)
**/
func (s *Message) Bytes() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Message) ToJson() et.Json {
	bt, err := s.Bytes()
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
* @param from et.Json, to []string, data et.Json
* @return *Message
**/
func newMessage(from et.Json, to []string) *Message {
	id := reg.UUID()
	return &Message{
		Created_at: time.Now(),
		Id:         id,
		From:       from,
		Channel:    "",
		To:         to,
		Ignored:    []string{},
		Data:       et.Json{},
		Message:    "",
	}
}

/**
* DecodeMessage
* @param messageType int, data []byte
* @return Message
**/
func DecodeMessage(data []byte) (Message, error) {
	var result Message
	err := json.Unmarshal(data, &result)
	if err != nil {
		return Message{}, err
	}

	return result, nil
}

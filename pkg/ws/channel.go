package ws

import "slices"

type TypeChannel string

const (
	TpQueue TypeChannel = "Queue"
	TpStack TypeChannel = "Stack"
	TpTopic TypeChannel = "Topic"
)

type Channel struct {
	Name        string      `json:"name"`
	Type        TypeChannel `json:"type"`
	Subscribers []string    `json:"subscribers"`
	Turn        int         `json:"turn"`
}

/**
* newChannel
* @param tp TypeChannel
* @return *Channel
**/
func newChannel(name string, tp TypeChannel) *Channel {
	return &Channel{
		Name:        name,
		Type:        tp,
		Subscribers: []string{},
		Turn:        0,
	}
}

/**
* subscriber
* @param subscriber string
**/
func (s *Channel) subscriber(subscriber string) {
	idx := slices.IndexFunc(s.Subscribers, func(item string) bool {
		return item == subscriber
	})
	if idx != -1 {
		return
	}

	s.Subscribers = append(s.Subscribers, subscriber)
}

/**
* remove
* @param subscriber string
**/
func (s *Channel) remove(subscriber string) {
	idx := slices.IndexFunc(s.Subscribers, func(item string) bool {
		return item == subscriber
	})
	if idx == -1 {
		return
	}

	s.Subscribers = append(s.Subscribers[:idx], s.Subscribers[idx+1:]...)
}

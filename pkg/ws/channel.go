package ws

import "slices"

type TypeChannel string

const (
	TpQueue TypeChannel = "Queue"
	TpTopic TypeChannel = "Topic"
)

type Channel struct {
	Type        TypeChannel `json:"type"`
	Subscribers []string    `json:"subscribers"`
	Turn        int         `json:"turn"`
}

/**
* addSubscriber
* @param subscriber string
**/
func (s *Channel) addSubscriber(subscriber string) {
	idx := slices.IndexFunc(s.Subscribers, func(item string) bool {
		return item == subscriber
	})
	if idx != -1 {
		return
	}

	s.Subscribers = append(s.Subscribers, subscriber)
}

/**
* removeSubscriber
* @param subscriber string
**/
func (s *Channel) removeSubscriber(subscriber string) {
	idx := slices.IndexFunc(s.Subscribers, func(item string) bool {
		return item == subscriber
	})
	if idx == -1 {
		return
	}

	s.Subscribers = append(s.Subscribers[:idx], s.Subscribers[idx+1:]...)
}

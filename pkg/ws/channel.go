package ws

type TypeChannel string

const (
	TpQueue TypeChannel = "Queue"
	TpTopic TypeChannel = "Topic"
)

type Channel struct {
	Type        TypeChannel `json:"type"`
	Subscribers []string    `json:"subscribers"`
}

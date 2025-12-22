package server

type Relation struct {
	Key    string `json:"key"`
	To     string `json:"to"`
	Column string `json:"column"`
}

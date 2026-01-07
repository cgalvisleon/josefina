package model

type Schema struct {
	Name   string            `json:"name"`
	Models map[string]*Model `json:"models"`
}

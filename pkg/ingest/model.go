package ingest

import "github.com/google/uuid"

func NewID() string {
	return uuid.NewString()
}

type Document struct {
	ID      string
	Content string
	Source  string
	Topic   string
}

type Chunk struct {
	ID      string
	Content string
	Topic   string
	Source  string
	Vector  []float32
}

package ingest

import (
	"strings"
)

type DefaultSplitter struct {
	MaxTokens int
	Overlap   int
}

func NewDefaultSplitter() *DefaultSplitter {
	return &DefaultSplitter{
		MaxTokens: 500,
		Overlap:   50,
	}
}

func (s *DefaultSplitter) Split(doc Document) ([]Chunk, error) {
	words := strings.Fields(doc.Content)
	var chunks []Chunk

	step := s.MaxTokens - s.Overlap

	for i := 0; i < len(words); i += step {
		end := i + s.MaxTokens
		if end > len(words) {
			end = len(words)
		}

		chunks = append(chunks, Chunk{
			ID:      NewID(),
			Content: strings.Join(words[i:end], " "),
			Topic:   doc.Topic,
			Source:  doc.Source,
		})
	}

	return chunks, nil
}

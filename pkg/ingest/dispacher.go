package ingest

import (
	"context"
	"io"

	"rag-system/internal/chunk"
	"rag-system/internal/embedding"
	"rag-system/internal/vector"
)

type Dispatcher struct {
	pdfParser   PDFParser
	splitter    chunk.Splitter
	embedder    embedding.Service
	vectorStore vector.Repository
}

func NewDispatcher(
	pdf PDFParser,
	splitter chunk.Splitter,
	embedder embedding.Service,
	vector vector.Repository,
) *Dispatcher {
	return &Dispatcher{
		pdfParser:   pdf,
		splitter:    splitter,
		embedder:    embedder,
		vectorStore: vector,
	}
}

func (d *Dispatcher) HandlePDF(
	ctx context.Context,
	reader io.Reader,
	filename string,
	topic string,
) error {

	docs, err := d.pdfParser.Parse(ctx, reader, filename, topic)
	if err != nil {
		return err
	}

	for _, doc := range docs {
		chunks, err := d.splitter.Split(doc)
		if err != nil {
			return err
		}

		for _, ch := range chunks {
			vec, err := d.embedder.Embed(ctx, ch.Content)
			if err != nil {
				return err
			}
			ch.Vector = vec
			err = d.vectorStore.Upsert(ctx, ch)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

package ingest

import (
	"context"
	"io"
)

type Service struct {
	dispatcher *Dispatcher
}

func NewService(d *Dispatcher) *Service {
	return &Service{dispatcher: d}
}

func (s *Service) IngestPDFAsync(
	ctx context.Context,
	reader io.Reader,
	filename string,
	topic string,
) error {
	go func() {
		_ = s.dispatcher.HandlePDF(ctx, reader, filename, topic)
	}()
	return nil
}

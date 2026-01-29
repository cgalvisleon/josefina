package ingest

import (
	"bytes"
	"context"
	"io"

	pdf "github.com/ledongthuc/pdf"
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(
	ctx context.Context,
	reader io.Reader,
	filename string,
	topic string,
) ([]Document, error) {

	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, reader)
	if err != nil {
		return nil, err
	}

	pdfReader, err := pdf.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, err
	}

	var text bytes.Buffer
	totalPage := pdfReader.NumPage()

	for i := 1; i <= totalPage; i++ {
		page := pdfReader.Page(i)
		content, _ := page.GetPlainText(nil)
		text.WriteString(content)
	}

	doc := Document{
		ID:      NewID(),
		Content: text.String(),
		Source:  filename,
		Topic:   topic,
	}

	return []Document{doc}, nil
}

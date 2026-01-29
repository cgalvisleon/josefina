package ingest

import (
	"context"
)

type Repository interface {
	Upsert(ctx context.Context, chunk Chunk) error
}

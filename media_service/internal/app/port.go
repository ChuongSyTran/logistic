package storage

import (
	"context"
	"io"
)

type FileStorage interface {
	Upload(ctx context.Context, file io.Reader, fileName string, folder string, prefix string) (string, string, string, error)
	Delete(ctx context.Context, publicID string) error
}

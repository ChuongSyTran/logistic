package app

import (
	"context"
	"io"
)

type FileStoragePort interface {
	Upload(ctx context.Context, file io.Reader, fileName string, folder string, prefix string) (string, string, string, error)
	Delete(ctx context.Context, publicID string) error
}

type FileStorage = FileStoragePort

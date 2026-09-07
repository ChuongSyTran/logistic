package app

import (
	"context"
	"io"
)

type MediaEngine interface {
	Upload(ctx context.Context, file io.Reader, fileName, folder, prefix string) (string, string, string, error)
	Delete(ctx context.Context, publicID string) error
}

type mediaEngineImpl struct {
	storage FileStoragePort
}

func NewMediaEngine(storage FileStoragePort) MediaEngine {
	return &mediaEngineImpl{storage: storage}
}

func (e *mediaEngineImpl) Upload(ctx context.Context, file io.Reader, fileName, folder, prefix string) (string, string, string, error) {
	return e.storage.Upload(ctx, file, fileName, folder, prefix)
}

func (e *mediaEngineImpl) Delete(ctx context.Context, publicID string) error {
	return e.storage.Delete(ctx, publicID)
}

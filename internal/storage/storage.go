package storage

import (
	"context"
	"io"
	"time"
)

type ObjectMetadata struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

type StorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
	GetMetadata(ctx context.Context, key string) (*ObjectMetadata, error)
}

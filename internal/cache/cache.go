package cache

import (
	"context"
	"io"
	"time"
)

var ErrNotFound = &NotFoundError{}

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}

type Cache interface {
	Close(ctx context.Context) error
	Get(ctx context.Context, key string) (string, time.Time, error)
	Set(ctx context.Context, key string, value string) error
	Iterate(ctx context.Context, fn IterateFunc) error
	Clear(ctx context.Context) error
	Delete(ctx context.Context, key string) error
	Reader(ctx context.Context, key string) (io.ReadCloser, error)
	Writer(ctx context.Context, key string) (io.WriteCloser, error)
}

type IterateFunc func(key string, age time.Time, value string) error

package cache

import (
	"context"
	"io"
	"time"

	"github.com/na4ma4/1password-direnv-tool/model"
)

var ErrNotFound = &NotFoundError{}

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "not found"
}

type Cache interface {
	Close(ctx context.Context) error
	Get(ctx context.Context, key string) (string, *model.FileList, time.Time, error)
	Set(ctx context.Context, key string, value string) (*model.FileList, error)
	Iterate(ctx context.Context, fn IterateFunc) error
	Clear(ctx context.Context) error
	Delete(ctx context.Context, key string) error
	Reader(ctx context.Context, key string) (io.ReadCloser, error)
	Writer(ctx context.Context, key string) (io.WriteCloser, error)
}

type IterateFunc func(key string, files *model.FileList, age time.Time, value string) error

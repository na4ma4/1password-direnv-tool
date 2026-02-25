package cache

import (
	"context"
	"io"
	"time"
)

type Noop struct{}

func NewNoop() *Noop {
	return &Noop{}
}

func (n *Noop) Close(_ context.Context) error {
	return nil
}

func (n *Noop) Get(_ context.Context, _ string) (string, time.Time, error) {
	return "", time.Time{}, ErrNotFound
}

func (n *Noop) Set(_ context.Context, _ string, _ string) error {
	return nil
}

func (n *Noop) Iterate(_ context.Context, _ IterateFunc) error {
	return nil
}

func (n *Noop) Clear(_ context.Context) error {
	return nil
}

func (n *Noop) Delete(_ context.Context, _ string) error {
	return nil
}

func (n *Noop) Reader(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, ErrNotFound
}

func (n *Noop) Writer(_ context.Context, _ string) (io.WriteCloser, error) {
	return noopWriteCloser{}, nil
}

type noopWriteCloser struct{}

func (noopWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (noopWriteCloser) Close() error {
	return nil
}

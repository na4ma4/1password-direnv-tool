package cache_test

import (
	"errors"
	"io"
	"testing"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
)

func TestDiskWriterReaderRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	d, err := cache.NewDisk(t.TempDir())
	if err != nil {
		t.Fatalf("NewDisk() error = %v", err)
	}
	defer d.Close(ctx)

	w, err := d.Writer(ctx, "roundtrip-key")
	if err != nil {
		t.Fatalf("Writer() error = %v", err)
	}

	_, err = w.Write([]byte("disk-value"))
	if err != nil {
		t.Fatalf("Writer().Write() error = %v", err)
	}

	err = w.Close()
	if err != nil {
		t.Fatalf("Writer().Close() error = %v", err)
	}

	r, err := d.Reader(ctx, "roundtrip-key")
	if err != nil {
		t.Fatalf("Reader() error = %v", err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(b) != "disk-value" {
		t.Fatalf("Reader() value = %q, want %q", string(b), "disk-value")
	}

	v, _, err := d.Get(ctx, "roundtrip-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if v != "disk-value" {
		t.Fatalf("Get() value = %q, want %q", v, "disk-value")
	}
}

func TestDiskMissingKeyReturnsNotFound(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	d, err := cache.NewDisk(t.TempDir())
	if err != nil {
		t.Fatalf("NewDisk() error = %v", err)
	}
	defer d.Close(ctx)

	_, readErr := d.Reader(ctx, "missing-key")
	if !errors.Is(readErr, cache.ErrNotFound) {
		t.Fatalf("Reader() error = %v, want %v", readErr, cache.ErrNotFound)
	}

	_, _, getErr := d.Get(ctx, "missing-key")
	if !errors.Is(getErr, cache.ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", getErr, cache.ErrNotFound)
	}
}

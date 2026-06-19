package cache_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
)

type fakeCache struct {
	getValue   string
	getAge     time.Time
	getErr     error
	setKey     string
	setValue   string
	setErr     error
	setCalls   int
	iterateErr error
	entries    []fakeEntry
}

type fakeEntry struct {
	key   string
	age   time.Time
	value string
}

func (f *fakeCache) Close(_ context.Context) error { return nil }

func (f *fakeCache) Get(_ context.Context, _ string) (string, time.Time, error) {
	return f.getValue, f.getAge, f.getErr
}

func (f *fakeCache) Set(_ context.Context, key string, value string) error {
	f.setCalls++
	f.setKey = key
	f.setValue = value
	return f.setErr
}

func (f *fakeCache) Iterate(_ context.Context, fn cache.IterateFunc) error {
	if f.iterateErr != nil {
		return f.iterateErr
	}

	for _, entry := range f.entries {
		if err := fn(entry.key, entry.age, entry.value); err != nil {
			return err
		}
	}

	return nil
}

func (f *fakeCache) Clear(_ context.Context) error { return nil }

func (f *fakeCache) Delete(_ context.Context, _ string) error { return nil }

func (f *fakeCache) Reader(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *fakeCache) Writer(_ context.Context, _ string) (io.WriteCloser, error) {
	return nopWriteCloser{}, nil
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }

func (nopWriteCloser) Close() error { return nil }

type fakeCodec struct {
	encodeErr error
	decodeErr error
}

func (f *fakeCodec) Encode(value string) (string, error) {
	if f.encodeErr != nil {
		return "", f.encodeErr
	}

	return "enc:" + value, nil
}

func (f *fakeCodec) Decode(value string) (string, error) {
	if f.decodeErr != nil {
		return "", f.decodeErr
	}

	return strings.TrimPrefix(value, "enc:"), nil
}

func (f *fakeCodec) ExportKey() (string, error) { return "", nil }

func (f *fakeCodec) ImportKey(_ any) error { return nil }

func TestEncryptionSetEncodesBeforeDelegating(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fc := &fakeCache{}
	ec := &fakeCodec{}
	e := cache.NewEncryption(fc, ec)

	if err := e.Set(ctx, "k", "plaintext"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if fc.setKey != "k" {
		t.Fatalf("Set() key = %q, want %q", fc.setKey, "k")
	}

	if fc.setValue != "enc:plaintext" {
		t.Fatalf("Set() value = %q, want %q", fc.setValue, "enc:plaintext")
	}
}

func TestEncryptionGetDecodesValue(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now().UTC()
	fc := &fakeCache{getValue: "enc:secret", getAge: now}
	ec := &fakeCodec{}
	e := cache.NewEncryption(fc, ec)

	got, age, err := e.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != "secret" {
		t.Fatalf("Get() value = %q, want %q", got, "secret")
	}

	if !age.Equal(now) {
		t.Fatalf("Get() age = %v, want %v", age, now)
	}
}

func TestEncryptionIterateDecodesEachValue(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now().UTC()
	fc := &fakeCache{
		entries: []fakeEntry{{key: "k1", age: now, value: "enc:v1"}, {key: "k2", age: now, value: "enc:v2"}},
	}
	ec := &fakeCodec{}
	e := cache.NewEncryption(fc, ec)

	seen := map[string]string{}
	err := e.Iterate(ctx, func(key string, _ time.Time, value string) error {
		seen[key] = value
		return nil
	})
	if err != nil {
		t.Fatalf("Iterate() error = %v", err)
	}

	if seen["k1"] != "v1" || seen["k2"] != "v2" {
		t.Fatalf("Iterate() decoded values = %#v, want k1=v1 and k2=v2", seen)
	}
}

func TestEncryptionReaderReturnsDecodedValue(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fc := &fakeCache{getValue: "enc:hello", getAge: time.Now().UTC()}
	ec := &fakeCodec{}
	e := cache.NewEncryption(fc, ec)

	r, err := e.Reader(ctx, "k")
	if err != nil {
		t.Fatalf("Reader() error = %v", err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(b) != "hello" {
		t.Fatalf("Reader() value = %q, want %q", string(b), "hello")
	}
}

func TestEncryptionWriterBuffersAndEncryptsOnClose(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fc := &fakeCache{}
	ec := &fakeCodec{}
	e := cache.NewEncryption(fc, ec)

	var w io.WriteCloser
	{
		var err error
		w, err = e.Writer(ctx, "key")
		if err != nil {
			t.Fatalf("Writer() error = %v", err)
		}
	}

	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatalf("Writer().Write() error = %v", err)
	}

	if _, err := w.Write([]byte(" world")); err != nil {
		t.Fatalf("Writer().Write() second call error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Writer().Close() error = %v", err)
	}

	if fc.setCalls != 1 {
		t.Fatalf("Writer().Close() set calls = %d, want 1", fc.setCalls)
	}

	if fc.setValue != "enc:hello world" {
		t.Fatalf("Writer().Close() value = %q, want %q", fc.setValue, "enc:hello world")
	}

	if _, err := w.Write([]byte("!")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Writer().Write() after close error = %v, want %v", err, io.ErrClosedPipe)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Writer().Close() second call error = %v", err)
	}

	if fc.setCalls != 1 {
		t.Fatalf("Writer().Close() second call set calls = %d, want 1", fc.setCalls)
	}
}

func TestEncryptionGetReturnsDecodeError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	failErr := errors.New("decode failed")
	fc := &fakeCache{getValue: "enc:value", getAge: time.Now().UTC()}
	ec := &fakeCodec{decodeErr: failErr}
	e := cache.NewEncryption(fc, ec)

	if _, _, err := e.Get(ctx, "k"); !errors.Is(err, failErr) {
		t.Fatalf("Get() error = %v, want %v", err, failErr)
	}
}

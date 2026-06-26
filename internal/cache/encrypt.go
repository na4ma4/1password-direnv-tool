package cache

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/na4ma4/1password-direnv-tool/internal/codec"
	"github.com/na4ma4/1password-direnv-tool/model"
)

type Encryption struct {
	cc    Cache
	codec codec.Codec
}

func NewEncryption(cc Cache, codec codec.Codec) *Encryption {
	return &Encryption{cc: cc, codec: codec}
}

func (e *Encryption) Close(ctx context.Context) error {
	return e.cc.Close(ctx)
}

// func (e *Encryption) Get(ctx context.Context, key string) (string, time.Time, error) {
// 	v, age, err := e.cc.Get(ctx, key)
// 	if err != nil {
// 		return "", time.Time{}, err
// 	}

// 	v, err = e.codec.Decode(v)
// 	if err != nil {
// 		return "", time.Time{}, err
// 	}

// 	return v, age, nil
// }

func (e *Encryption) Get(ctx context.Context, key string) (string, *model.FileList, time.Time, error) {
	v, files, age, err := e.cc.Get(ctx, key)
	if err != nil {
		return "", nil, time.Time{}, err
	}

	v, err = e.codec.Decode(v)
	if err != nil {
		return "", nil, time.Time{}, err
	}

	return v, files, age, nil
}

// func (e *Encryption) Set(ctx context.Context, key string, value string) error {
// 	v, err := e.codec.Encode(value)
// 	if err != nil {
// 		return err
// 	}

// 	return e.cc.Set(ctx, key, v)
// }

func (e *Encryption) Set(ctx context.Context, key string, value string) (*model.FileList, error) {
	v, err := e.codec.Encode(value)
	if err != nil {
		return nil, err
	}

	return e.cc.Set(ctx, key, v)
}

func (e *Encryption) Iterate(ctx context.Context, fn IterateFunc) error {
	return e.cc.Iterate(ctx, func(key string, files *model.FileList, age time.Time, value string) error {
		decoded, err := e.codec.Decode(value)
		if err != nil {
			return err
		}

		return fn(key, files, age, decoded)
	})
}

func (e *Encryption) Clear(ctx context.Context) error {
	return e.cc.Clear(ctx)
}

func (e *Encryption) Delete(ctx context.Context, key string) error {
	return e.cc.Delete(ctx, key)
}

func (e *Encryption) Reader(ctx context.Context, key string) (io.ReadCloser, error) {
	v, _, _, err := e.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewBufferString(v)), nil
}

func (e *Encryption) Writer(ctx context.Context, key string) (io.WriteCloser, error) {
	return &encryptionWriter{
		ctx: ctx,
		cc:  e,
		key: key,
	}, nil
}

type encryptionWriter struct {
	ctx    context.Context
	cc     *Encryption
	key    string
	buf    bytes.Buffer
	closed bool
}

func (w *encryptionWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}

	return w.buf.Write(p)
}

func (w *encryptionWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	_, err := w.cc.Set(w.ctx, w.key, w.buf.String())
	return err
}

package cache

import (
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/na4ma4/go-permbits"

	"github.com/na4ma4/1password-direnv-tool/model"
)

const (
	cacheFileExtension = ".dat"
	cacheFileMode      = 0o600
)

type Disk struct {
	path string
	root *os.Root
}

func NewDisk(path string) (*Disk, error) {
	path = os.ExpandEnv(path)

	_ = os.MkdirAll(path, 0o700)

	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}

	return &Disk{
		path: path,
		root: root,
	}, nil
}

func (d *Disk) Close(_ context.Context) error {
	return d.root.Close()
}

func (d *Disk) keyToFilename(key string) string {
	p := make([]string, 0, strings.Count(key, "/")+1)
	for s := range strings.SplitSeq(key, "/") {
		p = append(p,
			base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(s)),
		)
	}

	return filepath.Join(p...) + cacheFileExtension
}

func (d *Disk) Get(_ context.Context, key string) (string, *model.FileList, time.Time, error) {
	filename := d.keyToFilename(key)
	fileList := model.NewFileList(d.path)
	fileList.Append(filename)
	f, err := d.root.OpenFile(filename, os.O_RDONLY, cacheFileMode)
	if err != nil {
		return "", fileList, time.Time{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", fileList, time.Time{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}

	b, err := d.root.ReadFile(filename)
	if err != nil {
		return "", fileList, time.Time{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}

	return string(b), fileList, info.ModTime(), nil
}

func (d *Disk) Set(_ context.Context, key string, value string) (*model.FileList, error) {
	filename := d.keyToFilename(key)
	fileList := model.NewFileList(d.path)
	fileList.Append(filename)
	if err := os.MkdirAll(
		filepath.Join(d.path, filepath.Dir(filename)),
		permbits.UserAll,
	); err != nil {
		return fileList, err
	}
	f, err := d.root.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, cacheFileMode)
	if err != nil {
		return fileList, err
	}
	defer f.Close()

	_, err = f.WriteString(value)
	return fileList, err
}

func (d *Disk) Reader(_ context.Context, key string) (io.ReadCloser, error) {
	filename := d.keyToFilename(key)
	f, err := d.root.OpenFile(filename, os.O_RDONLY, cacheFileMode)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
	}

	return f, nil
}

func (d *Disk) Writer(_ context.Context, key string) (io.WriteCloser, error) {
	filename := d.keyToFilename(key)
	f, err := d.root.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, cacheFileMode)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (d *Disk) Clear(_ context.Context) error {
	defer func() { _ = os.Mkdir(d.path, 0o700) }()
	return os.RemoveAll(d.path)
}

func (d *Disk) Delete(_ context.Context, key string) error {
	filename := d.keyToFilename(key)
	return d.root.Remove(filename)
}

func (d *Disk) Iterate(ctx context.Context, fn IterateFunc) error {
	var entries []os.DirEntry
	{
		dir, err := d.root.Open(".")
		if err != nil {
			return err
		}
		defer dir.Close()

		entries, err = dir.ReadDir(-1)
		if err != nil {
			return err
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		var key string
		{
			keyBytes, err := base32.StdEncoding.
				WithPadding(base32.NoPadding).
				DecodeString(filename[:len(filename)-len(cacheFileExtension)])
			if err != nil {
				continue
			}
			key = string(keyBytes)
		}

		var value string
		var age time.Time
		var fileList *model.FileList
		{
			var err error
			value, fileList, age, err = d.Get(ctx, key)
			if err != nil {
				continue
			}
		}

		if err := fn(key, fileList, age, value); err != nil {
			if deleteErr, ok := errors.AsType[DeleteError](err); ok {
				_ = d.Delete(ctx, deleteErr.Key)
				continue
			}

			return err
		}
	}

	return nil
}

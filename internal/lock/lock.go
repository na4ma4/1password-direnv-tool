// Package lock provides file-based advisory locking used to serialize
// concurrent op-direnv invocations that would otherwise race to talk to the
// 1Password SDK / op CLI for the same item.
package lock

import (
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	lockFileExtension = ".lock"
	lockFileMode      = 0o600
	lockDirMode       = 0o700
	defaultRetry      = 100 * time.Millisecond
	globalLockName    = "op-direnv" + lockFileExtension
)

// Lock is an advisory lock that can be acquired and released.
type Lock interface {
	Acquire(ctx context.Context) error
	Release() error
}

// ForItem returns a file-backed lock keyed on the given item reference. The
// lock file lives inside dir; if dir is empty, [os.TempDir] is used.
func ForItem(dir, key string) (Lock, error) {
	return newFileLock(dir, keyToFilename(key))
}

// Global returns a single process-wide lock file in dir. Used when per-item
// keying isn't available (e.g. cache disabled).
func Global(dir string) (Lock, error) {
	return newFileLock(dir, globalLockName)
}

// Noop returns a lock that does nothing; useful when locking is disabled.
func Noop() Lock { return noopLock{} }

type noopLock struct{}

func (noopLock) Acquire(_ context.Context) error { return nil }
func (noopLock) Release() error                  { return nil }

type fileLock struct {
	flock *flock.Flock
	path  string
}

func newFileLock(dir, name string) (*fileLock, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	dir = os.ExpandEnv(dir)

	if err := os.MkdirAll(dir, lockDirMode); err != nil {
		return nil, fmt.Errorf("creating lock directory %q: %w", dir, err)
	}

	path := filepath.Join(dir, name)

	return &fileLock{
		flock: flock.New(path),
		path:  path,
	}, nil
}

func (f *fileLock) Acquire(ctx context.Context) error {
	ok, err := f.flock.TryLockContext(ctx, defaultRetry)
	if err != nil {
		return fmt.Errorf("acquiring lock %q: %w", f.path, err)
	}
	if !ok {
		return fmt.Errorf("acquiring lock %q: %w", f.path, errors.New("lock not acquired"))
	}
	_ = os.Chmod(f.path, lockFileMode)
	return nil
}

func (f *fileLock) Release() error {
	if err := f.flock.Unlock(); err != nil {
		return fmt.Errorf("releasing lock %q: %w", f.path, err)
	}
	return nil
}

func keyToFilename(key string) string {
	return base32.StdEncoding.
		WithPadding(base32.NoPadding).
		EncodeToString([]byte(key)) + lockFileExtension
}

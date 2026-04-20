package lock_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/na4ma4/1password-direnv-tool/internal/lock"
)

func TestForItemSerializesHolders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	l1, err := lock.ForItem(dir, "op://vault/item")
	if err != nil {
		t.Fatalf("ForItem #1: %v", err)
	}
	l2, err := lock.ForItem(dir, "op://vault/item")
	if err != nil {
		t.Fatalf("ForItem #2: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if aerr := l1.Acquire(ctx); aerr != nil {
		t.Fatalf("l1.Acquire: %v", aerr)
	}

	var acquired int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		if aerr := l2.Acquire(ctx); aerr != nil {
			t.Errorf("l2.Acquire: %v", aerr)
			return
		}
		atomic.StoreInt32(&acquired, 1)
		_ = l2.Release()
	}()

	time.Sleep(250 * time.Millisecond)
	if atomic.LoadInt32(&acquired) != 0 {
		t.Fatalf("second holder acquired lock while first still held it")
	}

	if rerr := l1.Release(); rerr != nil {
		t.Fatalf("l1.Release: %v", rerr)
	}

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("second holder never acquired lock: %v", ctx.Err())
	}

	if atomic.LoadInt32(&acquired) != 1 {
		t.Fatalf("second holder did not record acquisition")
	}
}

func TestForItemDifferentKeysDoNotBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	l1, err := lock.ForItem(dir, "op://vault/a")
	if err != nil {
		t.Fatalf("ForItem a: %v", err)
	}
	l2, err := lock.ForItem(dir, "op://vault/b")
	if err != nil {
		t.Fatalf("ForItem b: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if aerr := l1.Acquire(ctx); aerr != nil {
		t.Fatalf("l1.Acquire: %v", aerr)
	}
	defer func() { _ = l1.Release() }()

	if aerr := l2.Acquire(ctx); aerr != nil {
		t.Fatalf("l2.Acquire should succeed on different key: %v", aerr)
	}
	_ = l2.Release()
}

func TestNoop(t *testing.T) {
	t.Parallel()

	n := lock.Noop()
	if err := n.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := n.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

//go:build !cgo && !windows

package cache

import (
	"context"
	"errors"

	"github.com/1password/onepassword-sdk-go"
)

func onepasswordNewClient(ctx context.Context, accountName string) (*onepassword.Client, error) {
	return nil, errors.New("1Password client is not supported on this platform")
}

package opclient

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/viper"
)

const (
	// TODO bring this in from config or make it configurable via env var
	lazyTimeout = 10 * time.Second
)

// lazyOPClient is a struct that holds a lazily initialized 1Password client and a mutex
// for synchronizing access to it.
type lazyOPClient struct {
	lock   sync.Mutex
	client Client
}

// lazyClient is a global variable for lazy initialization of the 1Password client. It is used by the
// OnePasswordClientLazyInit function to ensure that the client is only initialized once and shared
// across the application.
//
//nolint:gochecknoglobals // lazyClient is intended to be a global variable for lazy initialization
var lazyClient = &lazyOPClient{}

type newClientResult struct {
	client Client
	err    error
}

// Func is a function type that returns a 1Password client. It is used to allow for
// lazy initialization of the client and to inject a mock client for testing.
type Func func(context.Context) (Client, error)

func OnePasswordClientLazyInit(_ context.Context, logger *slog.Logger) Func {
	accountName := viper.GetString("1password.account-name")

	if accountName == "" {
		accountName = "my"
	}

	return func(ctx context.Context) (Client, error) {
		lazyClient.lock.Lock()
		defer lazyClient.lock.Unlock()

		if lazyClient.client != nil {
			return lazyClient.client, nil
		}

		logger.DebugContext(ctx, "lazy initialization of 1Password client: started",
			slog.String("account_name", accountName),
		)
		defer logger.DebugContext(ctx, "lazy initialization of 1Password client: finished",
			slog.String("account_name", accountName),
		)

		{
			lazyCtx, cancel := context.WithTimeout(ctx, lazyTimeout)
			defer cancel()
			var err error
			lazyClient.client, err = NewSDKClient(lazyCtx, accountName)
			if err == nil {
				return lazyClient.client, nil
			}
			logger.ErrorContext(
				ctx,
				"failed to create 1Password SDK client, attempting to use CLI client",
				slogtool.ErrorAttr(err),
			)
		}

		var err error
		lazyClient.client, err = NewCLIClient(ctx, accountName)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create 1Password CLI client", slogtool.ErrorAttr(err))
			return nil, fmt.Errorf("failed to create 1Password client: %w", err)
		}

		return lazyClient.client, nil
	}
}

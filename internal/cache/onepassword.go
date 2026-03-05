package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/1password/onepassword-sdk-go"
	"github.com/dosquad/go-cliversion"
	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/viper"
)

// OPClientFunc is a function type that returns a 1Password client. It is used to allow for
// lazy initialization of the client and to inject a mock client for testing.
type OPClientFunc func(context.Context) (*onepassword.Client, error)

// lazyOPClient is a struct that holds a lazily initialized 1Password client and a mutex
// for synchronizing access to it.
type lazyOPClient struct {
	lock   sync.Mutex
	client *onepassword.Client
}

// lazyClient is a global variable for lazy initialization of the 1Password client. It is used by the
// OnePasswordClientLazyInit function to ensure that the client is only initialized once and shared
// across the application.
//
//nolint:gochecknoglobals // lazyClient is intended to be a global variable for lazy initialization
var lazyClient = &lazyOPClient{}

type newClientResult struct {
	client *onepassword.Client
	err    error
}

func newOnePasswordClientCancelable(ctx context.Context, accountName string) (*onepassword.Client, error) {
	resultCh := make(chan newClientResult, 1)

	go func() {
		client, err := onepassword.NewClient(
			ctx,
			onepassword.WithDesktopAppIntegration(accountName),
			onepassword.WithIntegrationInfo("1Password direnv tool", cliversion.Get().VersionString()),
		)

		resultCh <- newClientResult{client: client, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("creating 1Password client canceled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("creating 1Password client: %w", result.err)
		}

		return result.client, nil
	}
}

func OnePasswordClientLazyInit(_ context.Context, logger *slog.Logger) OPClientFunc {
	accountName := viper.GetString("1password.account-name")

	if accountName == "" {
		accountName = "my"
	}

	return func(ctx context.Context) (*onepassword.Client, error) {
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

		var err error
		lazyClient.client, err = newOnePasswordClientCancelable(ctx, accountName)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to create 1Password client", slogtool.ErrorAttr(err))
			return nil, err
		}

		return lazyClient.client, nil
	}
}

func jsonDecode[T any](data string) (T, error) {
	var v T
	err := json.NewDecoder(strings.NewReader(data)).Decode(&v)
	return v, err
}

func OnePasswordSecretResolve(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	opClient OPClientFunc,
	secretRef string,
) (string, error) {
	if cc != nil {
		cached, _, getErr := cc.Get(ctx, secretRef)
		switch {
		case getErr == nil:
			logger.DebugContext(ctx, "Cache hit for secret", slog.String("secret_ref", secretRef))
			return cached, nil
		case !errors.Is(getErr, ErrNotFound):
			logger.ErrorContext(ctx, "Cache error", slog.String("secret_ref", secretRef), slogtool.ErrorAttr(getErr))
		default:
			logger.DebugContext(ctx, "Cache miss for secret", slog.String("secret_ref", secretRef))
		}
	}

	var client *onepassword.Client
	{
		var err error
		client, err = opClient(ctx)
		if err != nil {
			return "", fmt.Errorf("initialising 1Password client: %w", err)
		}
	}

	var item string
	{
		var err error
		item, err = client.Secrets().Resolve(ctx, secretRef)
		if err != nil {
			return "", err
		}
	}

	if cc != nil {
		if err := cc.Set(ctx, secretRef, item); err != nil {
			logger.ErrorContext(ctx, "Cache error", slog.String("secret_ref", secretRef), slogtool.ErrorAttr(err))
		} else {
			logger.DebugContext(ctx, "Cached secret", slog.String("secret_ref", secretRef))
		}
	}

	return item, nil
}

func OnePasswordGetItem(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	opClient OPClientFunc,
	vaultID, itemID string,
) (onepassword.Item, error) {
	cacheKey := "item:" + vaultID + ":" + itemID
	if cc != nil { //nolint:nestif // nesting is acceptable here for cache retrieval logic
		if cached, _, getErr := cc.Get(ctx, cacheKey); getErr == nil {
			logger.DebugContext(ctx, "Cache hit for item", slog.String("cache_key", cacheKey))
			item, err := jsonDecode[onepassword.Item](cached)
			if err != nil {
				logger.WarnContext(ctx, "Cache value has unexpected type",
					slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err),
				)
			} else {
				return item, nil
			}
			logger.WarnContext(ctx, "Cache value has unexpected type", slog.String("cache_key", cacheKey))
		} else if !errors.Is(getErr, ErrNotFound) {
			logger.ErrorContext(ctx, "Cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(getErr))
		} else {
			logger.DebugContext(ctx, "Cache miss for item", slog.String("cache_key", cacheKey))
		}
	}

	logger.DebugContext(ctx, "initialising 1Password client")
	var client *onepassword.Client
	{
		var err error
		client, err = opClient(ctx)
		if err != nil {
			return onepassword.Item{}, fmt.Errorf("initialising 1Password client: %w", err)
		}
	}

	logger.DebugContext(ctx, "retrieving item")
	var item onepassword.Item
	{
		var err error
		item, err = client.Items().Get(ctx, vaultID, itemID)
		if err != nil {
			return onepassword.Item{}, err
		}
	}

	if cc != nil {
		logger.DebugContext(ctx, "saving item to cache", slog.String("cache_key", cacheKey))
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(item); err != nil {
			logger.ErrorContext(ctx, "Failed to encode item for caching",
				slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err),
			)
			return item, nil
		}

		if err := cc.Set(ctx, cacheKey, buf.String()); err != nil {
			logger.ErrorContext(ctx, "Cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
		} else {
			logger.DebugContext(ctx, "Cached item", slog.String("cache_key", cacheKey))
		}
	}

	return item, nil
}

func OnePasswordVaultList(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	opClient OPClientFunc,
) ([]onepassword.VaultOverview, error) {
	cacheKey := "vaults:list"
	if cc != nil { //nolint:nestif // nesting is acceptable here for cache retrieval logic
		if cached, _, getErr := cc.Get(ctx, cacheKey); getErr == nil {
			logger.DebugContext(ctx, "Cache hit for vault list", slog.String("cache_key", cacheKey))
			vaults, err := jsonDecode[[]onepassword.VaultOverview](cached)
			if err == nil {
				return vaults, nil
			}
		} else if !errors.Is(getErr, ErrNotFound) {
			logger.ErrorContext(ctx, "Cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(getErr))
		} else {
			logger.DebugContext(ctx, "Cache miss for vault list", slog.String("cache_key", cacheKey))
		}
	}

	logger.DebugContext(ctx, "initialising 1password client")
	var client *onepassword.Client
	{
		var err error
		client, err = opClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("initialising 1Password client: %w", err)
		}
	}

	logger.DebugContext(ctx, "retrieving vaults list")
	var vaults []onepassword.VaultOverview
	{
		var err error
		vaults, err = client.Vaults().List(ctx)
		if err != nil {
			return nil, err
		}
	}

	if cc != nil {
		logger.DebugContext(ctx, "saving vault list to cache", slog.String("cache_key", cacheKey))
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(vaults); err != nil {
			logger.ErrorContext(ctx, "failed to encode vault list for caching",
				slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
			return vaults, nil
		}
		if err := cc.Set(ctx, cacheKey, buf.String()); err != nil {
			logger.ErrorContext(ctx, "cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
		} else {
			logger.DebugContext(ctx, "cached vault list", slog.String("cache_key", cacheKey))
		}
	}

	return vaults, nil
}

func OnePasswordItemList(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	opClient OPClientFunc,
	vaultID string,
) ([]onepassword.ItemOverview, error) {
	cacheKey := "items:list:" + vaultID
	if cc != nil { //nolint:nestif // nesting is acceptable here for cache retrieval logic
		if cached, _, getErr := cc.Get(ctx, cacheKey); getErr == nil {
			logger.DebugContext(ctx, "Cache hit for item list", slog.String("cache_key", cacheKey))
			items, err := jsonDecode[[]onepassword.ItemOverview](cached)
			if err == nil {
				return items, nil
			}
			logger.WarnContext(ctx, "Cache value has unexpected type",
				slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
		} else if !errors.Is(getErr, ErrNotFound) {
			logger.ErrorContext(ctx, "Cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(getErr))
		} else {
			logger.DebugContext(ctx, "Cache miss for item list", slog.String("cache_key", cacheKey))
		}
	}

	logger.DebugContext(ctx, "initialising 1password client")
	var client *onepassword.Client
	{
		var err error
		client, err = opClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("initialising 1Password client: %w", err)
		}
	}

	logger.DebugContext(ctx, "retrieving item list")
	var items []onepassword.ItemOverview
	{
		var err error
		items, err = client.Items().List(ctx, vaultID)
		if err != nil {
			return nil, err
		}
	}

	if cc != nil {
		logger.DebugContext(ctx, "saving item list to cache", slog.String("cache_key", cacheKey))
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(items); err != nil {
			logger.ErrorContext(ctx, "Failed to encode item list for caching",
				slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
			return items, nil
		}

		if err := cc.Set(ctx, cacheKey, buf.String()); err != nil {
			logger.ErrorContext(ctx, "Cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
		} else {
			logger.DebugContext(ctx, "Cached item list", slog.String("cache_key", cacheKey))
		}
	}

	return items, nil
}

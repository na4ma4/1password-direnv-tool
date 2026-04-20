package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/1password/onepassword-sdk-go"
	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/viper"

	"github.com/na4ma4/1password-direnv-tool/internal/opclient"
)

func jsonDecode[T any](data string) (T, error) {
	var v T
	err := json.NewDecoder(strings.NewReader(data)).Decode(&v)
	return v, err
}

// isTimeoutError checks if an error is a timeout error.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	errStr := err.Error()
	return strings.Contains(errStr, "canceled") &&
		(strings.Contains(errStr, "DeadlineExceeded") ||
			strings.Contains(errStr, "context deadline exceeded"))
}

// resolveSecretCancelable wraps the secret resolution with explicit timeout handling.
// This ensures the call returns even if the 1Password SDK doesn't properly respect context cancellation.
func resolveSecretCancelable(
	ctx context.Context,
	client opclient.Client,
	secretRef string,
) (string, error) {
	type result struct {
		value string
		err   error
	}

	resultCh := make(chan result, 1)

	go func() {
		value, err := client.Secrets().Resolve(ctx, secretRef)
		resultCh <- result{value: value, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("resolving secret %q canceled: %w", secretRef, ctx.Err())
	case res := <-resultCh:
		return res.value, res.err
	}
}

// resolveSecretWithFallback resolves a secret using the socket API, with fallback to op CLI on timeout.
func resolveSecretWithFallback(
	ctx context.Context,
	client opclient.Client,
	secretRef string,
	logger *slog.Logger,
) (string, error) {
	value, err := resolveSecretCancelable(ctx, client, secretRef)
	if err != nil && isTimeoutError(err) {
		logger.WarnContext(ctx, "socket API timeout, falling back to op CLI",
			slog.String("secret_ref", secretRef),
			slog.String("error", err.Error()),
		)
		return resolveSecretViaCLI(ctx, secretRef, logger)
	}
	return value, err
}

// getItemCancelable wraps the item retrieval with explicit timeout handling.
func getItemCancelable(
	ctx context.Context,
	client opclient.Client,
	vaultID, itemID string,
) (onepassword.Item, error) {
	type result struct {
		item onepassword.Item
		err  error
	}

	resultCh := make(chan result, 1)

	go func() {
		item, err := client.Items().Get(ctx, vaultID, itemID)
		resultCh <- result{item: item, err: err}
	}()

	select {
	case <-ctx.Done():
		return onepassword.Item{}, fmt.Errorf("getting item %q from vault %q canceled: %w", itemID, vaultID, ctx.Err())
	case res := <-resultCh:
		return res.item, res.err
	}
}

// getItemWithFallback gets an item using the socket API, with fallback to op CLI on timeout.
func getItemWithFallback(
	ctx context.Context,
	client opclient.Client,
	vaultID, itemID string,
	logger *slog.Logger,
) (onepassword.Item, error) {
	item, err := getItemCancelable(ctx, client, vaultID, itemID)
	if err != nil && isTimeoutError(err) {
		logger.WarnContext(ctx, "socket API timeout, falling back to op CLI",
			slog.String("vault_id", vaultID),
			slog.String("item_id", itemID),
			slog.String("error", err.Error()),
		)

		return getItemViaCLI(ctx, vaultID, itemID, logger)
	} else if err != nil {
		logger.WarnContext(ctx, "socket API error, not a timeout, returning error",
			slog.String("vault_id", vaultID),
			slog.String("item_id", itemID),
			slog.String("error", err.Error()),
		)

		return onepassword.Item{}, err
	}
	return item, err
}

// listVaultsCancelable wraps the vault list retrieval with explicit timeout handling.
func listVaultsCancelable(
	ctx context.Context,
	client opclient.Client,
) ([]onepassword.VaultOverview, error) {
	type result struct {
		vaults []onepassword.VaultOverview
		err    error
	}

	resultCh := make(chan result, 1)

	go func() {
		vaults, err := client.Vaults().List(ctx)
		resultCh <- result{vaults: vaults, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("listing vaults canceled: %w", ctx.Err())
	case res := <-resultCh:
		return res.vaults, res.err
	}
}

// listVaultsWithFallback lists vaults using the socket API, with fallback to op CLI on timeout.
func listVaultsWithFallback(
	ctx context.Context,
	client opclient.Client,
	logger *slog.Logger,
) ([]onepassword.VaultOverview, error) {
	vaults, err := listVaultsCancelable(ctx, client)
	if err != nil && isTimeoutError(err) {
		logger.WarnContext(ctx, "socket API timeout, falling back to op CLI",
			slog.String("error", err.Error()),
		)
		return listVaultsViaCLI(ctx, logger)
	}
	return vaults, err
}

// listItemsCancelable wraps the item list retrieval with explicit timeout handling.
func listItemsCancelable(
	ctx context.Context,
	client opclient.Client,
	vaultID string,
) ([]onepassword.ItemOverview, error) {
	type result struct {
		items []onepassword.ItemOverview
		err   error
	}

	resultCh := make(chan result, 1)

	go func() {
		items, err := client.Items().List(ctx, vaultID)
		resultCh <- result{items: items, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("listing items in vault %q canceled: %w", vaultID, ctx.Err())
	case res := <-resultCh:
		return res.items, res.err
	}
}

// listItemsWithFallback lists items using the socket API, with fallback to op CLI on timeout.
func listItemsWithFallback(
	ctx context.Context,
	client opclient.Client,
	vaultID string,
	logger *slog.Logger,
) ([]onepassword.ItemOverview, error) {
	items, err := listItemsCancelable(ctx, client, vaultID)
	if err != nil && isTimeoutError(err) {
		logger.WarnContext(ctx, "socket API timeout, falling back to op CLI",
			slog.String("vault_id", vaultID),
			slog.String("error", err.Error()),
		)
		return listItemsViaCLI(ctx, vaultID, logger)
	}
	return items, err
}

// resolveSecretViaCLI resolves a secret reference using the op CLI tool.
func resolveSecretViaCLI(ctx context.Context, secretRef string, logger *slog.Logger) (string, error) {
	logger.InfoContext(ctx, "using op CLI for secret resolution", slog.String("secret_ref", secretRef))

	accountName := viper.GetString("1password.account-name")
	if accountName == "" {
		accountName = "my"
	}

	args := []string{"read", secretRef, "--no-newline"}
	if accountName != "" {
		args = append([]string{"--account", accountName}, args...)
	}

	cmd := exec.CommandContext(ctx, "op", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		return "", fmt.Errorf("op read failed after %v: %w (stderr: %s)", duration, err, stderr.String())
	}

	result := strings.TrimSpace(stdout.String())
	logger.DebugContext(ctx, "op CLI secret resolution successful",
		slog.String("secret_ref", secretRef),
		slog.Duration("duration", duration),
	)

	return result, nil
}

// getItemViaCLI retrieves an item using the op CLI tool.
func getItemViaCLI(ctx context.Context, vaultID, itemID string, logger *slog.Logger) (onepassword.Item, error) {
	logger.InfoContext(ctx, "using op CLI for item retrieval",
		slog.String("vault_id", vaultID),
		slog.String("item_id", itemID),
	)

	accountName := viper.GetString("1password.account-name")
	if accountName == "" {
		accountName = "my"
	}

	args := []string{"item", "get", itemID, "--vault", vaultID, "--format", "json"}
	if accountName != "" {
		args = append([]string{"--account", accountName}, args...)
	}

	cmd := exec.CommandContext(ctx, "op", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	if runErr != nil {
		return onepassword.Item{}, fmt.Errorf("failed op item get after %v: %w", duration, runErr)
	}

	var item onepassword.Item
	if decodeErr := json.NewDecoder(&stdout).Decode(&item); decodeErr != nil {
		return onepassword.Item{}, fmt.Errorf("failed to decode op item get response: %w", decodeErr)
	}

	logger.DebugContext(ctx, "op CLI item retrieval successful",
		slog.String("vault_id", vaultID),
		slog.String("item_id", itemID),
		slog.Duration("duration", duration),
	)

	return item, nil
}

// listVaultsViaCLI lists vaults using the op CLI tool.
func listVaultsViaCLI(ctx context.Context, logger *slog.Logger) ([]onepassword.VaultOverview, error) {
	logger.InfoContext(ctx, "using op CLI for vault list")

	accountName := viper.GetString("1password.account-name")
	if accountName == "" {
		accountName = "my"
	}

	args := []string{"vault", "list", "--format", "json"}
	if accountName != "" {
		args = append([]string{"--account", accountName}, args...)
	}

	cmd := exec.CommandContext(ctx, "op", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	if runErr != nil {
		return nil, fmt.Errorf("op vault list failed after %v: %w (stderr: %s)", duration, runErr, stderr.String())
	}

	var vaults []onepassword.VaultOverview
	if decodeErr := json.NewDecoder(&stdout).Decode(&vaults); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode op vault list response: %w", decodeErr)
	}

	logger.DebugContext(ctx, "op CLI vault list successful",
		slog.Int("vault_count", len(vaults)),
		slog.Duration("duration", duration),
	)

	return vaults, nil
}

// listItemsViaCLI lists items in a vault using the op CLI tool.
func listItemsViaCLI(ctx context.Context, vaultID string, logger *slog.Logger) ([]onepassword.ItemOverview, error) {
	logger.InfoContext(ctx, "using op CLI for item list", slog.String("vault_id", vaultID))

	accountName := viper.GetString("1password.account-name")
	if accountName == "" {
		accountName = "my"
	}

	args := []string{"item", "list", "--vault", vaultID, "--format", "json"}
	if accountName != "" {
		args = append([]string{"--account", accountName}, args...)
	}

	cmd := exec.CommandContext(ctx, "op", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	if runErr != nil {
		return nil, fmt.Errorf("op item list failed after %v: %w (stderr: %s)", duration, runErr, stderr.String())
	}

	var items []onepassword.ItemOverview
	if decodeErr := json.NewDecoder(&stdout).Decode(&items); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode op item list response: %w", decodeErr)
	}

	logger.DebugContext(ctx, "op CLI item list successful",
		slog.String("vault_id", vaultID),
		slog.Int("item_count", len(items)),
		slog.Duration("duration", duration),
	)

	return items, nil
}

func OnePasswordSecretResolve(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	opClient opclient.Func,
	secretRef string,
) (string, error) {
	if cc != nil {
		cached, _, getErr := cc.Get(ctx, secretRef)
		switch {
		case getErr == nil:
			logger.DebugContext(ctx, "cache hit for secret", slog.String("secret_ref", secretRef))
			return cached, nil
		case !errors.Is(getErr, ErrNotFound):
			logger.ErrorContext(ctx, "cache error", slog.String("secret_ref", secretRef), slogtool.ErrorAttr(getErr))
		default:
			logger.DebugContext(ctx, "cache miss for secret", slog.String("secret_ref", secretRef))
		}
	}

	var client opclient.Client
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
		item, err = resolveSecretWithFallback(ctx, client, secretRef, logger)
		if err != nil {
			return "", err
		}
	}

	if cc != nil {
		if err := cc.Set(ctx, secretRef, item); err != nil {
			logger.ErrorContext(ctx, "cache error", slog.String("secret_ref", secretRef), slogtool.ErrorAttr(err))
		} else {
			logger.DebugContext(ctx, "cached secret", slog.String("secret_ref", secretRef))
		}
	}

	return item, nil
}

func OnePasswordGetItem(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	opClient opclient.Func,
	vaultID, itemID string,
) (onepassword.Item, error) {
	cacheKey := "item:" + vaultID + ":" + itemID
	if cc != nil { //nolint:nestif // nesting is acceptable here for cache retrieval logic
		if cached, _, getErr := cc.Get(ctx, cacheKey); getErr == nil {
			logger.DebugContext(ctx, "cache hit for item", slog.String("cache_key", cacheKey))
			item, err := jsonDecode[onepassword.Item](cached)
			if err != nil {
				logger.WarnContext(ctx, "cache value has unexpected type",
					slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err),
				)
			} else {
				return item, nil
			}
			logger.WarnContext(ctx, "cache value has unexpected type", slog.String("cache_key", cacheKey))
		} else if !errors.Is(getErr, ErrNotFound) {
			logger.ErrorContext(ctx, "cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(getErr))
		} else {
			logger.DebugContext(ctx, "cache miss for item", slog.String("cache_key", cacheKey))
		}
	}

	logger.DebugContext(ctx, "initialising 1Password client")
	var client opclient.Client
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
		item, err = getItemWithFallback(ctx, client, vaultID, itemID, logger)
		if err != nil {
			return onepassword.Item{}, err
		}
	}

	if cc != nil {
		logger.DebugContext(ctx, "saving item to cache", slog.String("cache_key", cacheKey))
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(item); err != nil {
			logger.ErrorContext(ctx, "failed to encode item for caching",
				slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err),
			)
			return item, nil
		}

		if err := cc.Set(ctx, cacheKey, buf.String()); err != nil {
			logger.ErrorContext(ctx, "cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
		} else {
			logger.DebugContext(ctx, "cached item", slog.String("cache_key", cacheKey))
		}
	}

	return item, nil
}

func OnePasswordVaultList(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	opClient opclient.Func,
) ([]onepassword.VaultOverview, error) {
	cacheKey := "vaults:list"
	if cc != nil { //nolint:nestif // nesting is acceptable here for cache retrieval logic
		if cached, _, getErr := cc.Get(ctx, cacheKey); getErr == nil {
			logger.DebugContext(ctx, "cache hit for vault list", slog.String("cache_key", cacheKey))
			vaults, err := jsonDecode[[]onepassword.VaultOverview](cached)
			if err == nil {
				return vaults, nil
			}
		} else if !errors.Is(getErr, ErrNotFound) {
			logger.ErrorContext(ctx, "cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(getErr))
		} else {
			logger.DebugContext(ctx, "cache miss for vault list", slog.String("cache_key", cacheKey))
		}
	}

	logger.DebugContext(ctx, "initialising 1password client")
	var client opclient.Client
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
		vaults, err = listVaultsWithFallback(ctx, client, logger)
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
	opClient opclient.Func,
	vaultID string,
) ([]onepassword.ItemOverview, error) {
	cacheKey := "items:list:" + vaultID
	if cc != nil { //nolint:nestif // nesting is acceptable here for cache retrieval logic
		if cached, _, getErr := cc.Get(ctx, cacheKey); getErr == nil {
			logger.DebugContext(ctx, "cache hit for item list", slog.String("cache_key", cacheKey))
			items, err := jsonDecode[[]onepassword.ItemOverview](cached)
			if err == nil {
				return items, nil
			}
			logger.WarnContext(ctx, "cache value has unexpected type",
				slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
		} else if !errors.Is(getErr, ErrNotFound) {
			logger.ErrorContext(ctx, "cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(getErr))
		} else {
			logger.DebugContext(ctx, "cache miss for item list", slog.String("cache_key", cacheKey))
		}
	}

	logger.DebugContext(ctx, "initialising 1password client")
	var client opclient.Client
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
		items, err = listItemsWithFallback(ctx, client, vaultID, logger)
		if err != nil {
			return nil, err
		}
	}

	if cc != nil {
		logger.DebugContext(ctx, "saving item list to cache", slog.String("cache_key", cacheKey))
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(items); err != nil {
			logger.ErrorContext(ctx, "failed to encode item list for caching",
				slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
			return items, nil
		}

		if err := cc.Set(ctx, cacheKey, buf.String()); err != nil {
			logger.ErrorContext(ctx, "cache error", slog.String("cache_key", cacheKey), slogtool.ErrorAttr(err))
		} else {
			logger.DebugContext(ctx, "cached item list", slog.String("cache_key", cacheKey))
		}
	}

	return items, nil
}

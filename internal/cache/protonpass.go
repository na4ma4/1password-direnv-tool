package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
)

type ProtonPassRunner func(ctx context.Context, args ...string) (string, error)

type ProtonPassDownloader func(ctx context.Context, shareID, itemID, attachmentID string) ([]byte, error)

func ProtonPassVaultList(ctx context.Context, cc Cache, logger *slog.Logger, runner ProtonPassRunner) (string, error) {
	const cacheKey = "protonpass/vaults/list"

	if cached, _, err := cc.Get(ctx, cacheKey); err == nil {
		logger.DebugContext(ctx, "cache hit for vault list")
		return cached, nil
	} else if errors.Is(err, ErrNotFound) {
		logger.DebugContext(ctx, "cache miss for vault list")
	} else {
		logger.WarnContext(ctx, "cache error for vault list", slog.String("error", err.Error()))
	}

	var data string
	{
		var err error
		data, err = runner(ctx, "vault", "list", "--output", "json")
		if err != nil {
			return "", err
		}
	}

	if err := cc.Set(ctx, cacheKey, data); err != nil {
		logger.WarnContext(ctx, "failed to cache vault list", slog.String("error", err.Error()))
	}

	return data, nil
}

func ProtonPassViewItem(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	runner ProtonPassRunner,
	shareID, itemRef string,
) (string, error) {
	cacheKey := "protonpass/item_view/" + shareID + "/" + itemRef

	if cached, _, err := cc.Get(ctx, cacheKey); err == nil {
		logger.DebugContext(ctx, "cache hit for item view", slog.String("item", itemRef))
		return cached, nil
	} else if errors.Is(err, ErrNotFound) {
		logger.DebugContext(ctx, "cache miss for item view", slog.String("item", itemRef))
	} else {
		logger.WarnContext(ctx, "cache error for item view", slog.String("error", err.Error()))
	}

	itemURI := "pass://" + shareID + "/" + itemRef
	var data string
	{
		var err error
		data, err = runner(ctx, "item", "view", itemURI, "--output", "json")
		if err != nil {
			return "", err
		}
	}

	if err := cc.Set(ctx, cacheKey, data); err != nil {
		logger.WarnContext(ctx, "failed to cache item view", slog.String("error", err.Error()))
	}

	return data, nil
}

func ProtonPassResolveSecret(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	runner ProtonPassRunner,
	ref string,
) (string, error) {
	if cached, _, err := cc.Get(ctx, ref); err == nil {
		logger.DebugContext(ctx, "cache hit for secret", slog.String("ref", ref))
		return cached, nil
	} else if errors.Is(err, ErrNotFound) {
		logger.DebugContext(ctx, "cache miss for secret", slog.String("ref", ref))
	} else {
		logger.WarnContext(ctx, "cache error for secret", slog.String("error", err.Error()))
	}

	var data string
	{
		var err error
		data, err = runner(ctx, "item", "view", ref)
		if err != nil {
			return "", err
		}
	}

	if err := cc.Set(ctx, ref, data); err != nil {
		logger.WarnContext(ctx, "failed to cache secret", slog.String("error", err.Error()))
	}

	return data, nil
}

func ProtonPassDownloadAttachment(
	ctx context.Context,
	cc Cache,
	logger *slog.Logger,
	downloader ProtonPassDownloader,
	shareID, itemID, attachmentID string,
) ([]byte, error) {
	cacheKey := "protonpass/attachment/" + shareID + "/" + itemID + "/" + attachmentID

	if cached, _, cachedErr := cc.Get(ctx, cacheKey); cachedErr == nil {
		var data []byte
		err := json.Unmarshal([]byte(cached), &data)
		if err == nil {
			logger.DebugContext(ctx, "cache hit for attachment", slog.String("attachment_id", attachmentID))
			return data, nil
		}
		logger.WarnContext(ctx, "failed to decode cached attachment", slog.String("error", err.Error()))
	} else if errors.Is(cachedErr, ErrNotFound) {
		logger.DebugContext(ctx, "cache miss for attachment", slog.String("attachment_id", attachmentID))
	} else {
		logger.WarnContext(ctx, "cache error for attachment", slog.String("error", cachedErr.Error()))
	}

	var data []byte
	{
		var err error
		data, err = downloader(ctx, shareID, itemID, attachmentID)
		if err != nil {
			return nil, err
		}
	}

	var encoded []byte
	{
		var err error
		encoded, err = json.Marshal(data)
		if err != nil {
			logger.WarnContext(ctx, "failed to encode attachment for cache", slog.String("error", err.Error()))
			return data, nil
		}
	}

	if err := cc.Set(ctx, cacheKey, string(encoded)); err != nil {
		logger.WarnContext(ctx, "failed to cache attachment", slog.String("error", err.Error()))
	}

	return data, nil
}

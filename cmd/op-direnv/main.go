package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
	"github.com/na4ma4/1password-direnv-tool/internal/codec"
	"github.com/na4ma4/1password-direnv-tool/internal/itemref"
	"github.com/na4ma4/1password-direnv-tool/internal/lazy"
	"github.com/na4ma4/1password-direnv-tool/internal/openv"
	"github.com/na4ma4/1password-direnv-tool/model"
	onepassword "github.com/na4ma4/1password-direnv-tool/providers/1password"
	protonpass "github.com/na4ma4/1password-direnv-tool/providers/protonpass"
)

const (
	defaultTimeout = 2 * time.Minute
)

func init() {
	rootCmd.PersistentFlags().StringP("section", "s", "Environment", "Section name containing environment variables")
	_ = viper.BindPFlag("section", rootCmd.PersistentFlags().Lookup("section"))
	_ = viper.BindEnv("section", "OP_SECTION")

	rootCmd.PersistentFlags().DurationP("timeout", "t", defaultTimeout, "Timeout for operations")
	_ = viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
}

func setupCache(ctx context.Context, logger *slog.Logger) (cache.Cache, error) {
	if !viper.GetBool("cache.enabled") {
		logger.DebugContext(ctx, "caching disabled")
		return cache.NewNoop(), nil
	}

	cachePath := viper.GetString("cache.path")
	cst, err := cache.NewDisk(cachePath)
	if err != nil {
		logger.ErrorContext(ctx, "failed to initialize file cache", slogtool.ErrorAttr(err))
		return nil, fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	if err = cst.Iterate(ctx, cache.ExpireFunc(viper.GetDuration("cache.age"))); err != nil {
		logger.ErrorContext(ctx, "failed to expire old cache entries", slogtool.ErrorAttr(err))
		return nil, fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	logger.DebugContext(ctx, "initialized file cache",
		slog.String("cache_path", cachePath),
		slog.Duration("cache_age", viper.GetDuration("cache.age")),
	)

	return cache.NewEncryption(cst, codec.Default), nil
}

func getProvider(
	ctx context.Context,
	logger *slog.Logger,
	scheme string,
	itemRef itemref.Ref,
	cst cache.Cache,
) (model.Provider, error) {
	switch scheme {
	case "pass":
		return protonpass.NewProvider(logger, cst), nil
	case "op":
		client := cache.OnePasswordClientLazyInit(ctx, logger)
		return onepassword.NewProvider(client, cst, logger), nil
	default:
		logger.ErrorContext(ctx, "unsupported provider scheme in item reference",
			slog.String("scheme", scheme),
		)
		return nil, fmt.Errorf("%w: unsupported provider scheme %q in item reference %q",
			cmdconst.ErrNoUsage, scheme, itemRef.String())
	}
}

func setupLogger() *slog.Logger {
	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	logger.Debug("enabled debug logging")

	return logger
}

func outputEnvVars(envVars <-chan model.EnvVar) {
	watchList := model.NewFileList("")

	fmt.Fprintln(os.Stdout, "## Exports")
	for env := range envVars {
		//nolint:gosec,nolintlint
		fmt.Fprintf(os.Stdout, "export %s=%s\n", env.GetName(), shellQuote(env.GetValue()))
		watchList.Merge(env.GetFileList())
	}

	if watchList.Len() > 0 {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "## Direnv watch files")
		for _, file := range watchList.GetFiles() {
			//nolint:gosec,nolintlint
			fmt.Fprintf(os.Stdout, "watch_file %s\n", shellQuote(file))
		}
	}
}

func mainCmd(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	ctx, cancel := context.WithTimeout(ctx, viper.GetDuration("timeout"))
	defer cancel()

	logger := setupLogger()

	if codec.Default == nil {
		logger.ErrorContext(ctx, "no codec available for decrypting item reference")
		return fmt.Errorf("%w%v", cmdconst.ErrNoUsage, "no codec available for decrypting item reference")
	}

	cst, err := setupCache(ctx, logger)
	if err != nil {
		return err
	}

	itemRef, err := itemref.GetRef(ctx, codec.Default, nil)
	if err != nil || itemRef.IsEmpty() {
		logger.ErrorContext(ctx, "failed to get item reference from configuration", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	logger.InfoContext(ctx, "loading environment variables", slog.String("item", itemRef.String()))

	refURL, err := url.Parse(itemRef.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse item reference URI",
			slog.String("ref", itemRef.String()),
			slogtool.ErrorAttr(err),
		)
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	scheme := refURL.Scheme

	provider, err := getProvider(ctx, logger, scheme, itemRef, cst)
	if err != nil {
		return err
	}

	var opResolver, passResolver model.SecretResolver

	switch scheme {
	case "op":
		opResolver = provider.SecretResolver()
		passResolver = lazy.NewResolver(func() (model.SecretResolver, error) {
			p := protonpass.NewProvider(logger, cst)
			return p.SecretResolver(), nil
		})
	case "pass":
		opClient := cache.OnePasswordClientLazyInit(ctx, logger)
		opResolver = lazy.NewResolver(func() (model.SecretResolver, error) {
			p := onepassword.NewProvider(opClient, cst, logger)
			return p.SecretResolver(), nil
		})
		passResolver = provider.SecretResolver()
	}

	section := viper.GetString("section")
	ope := openv.New(provider, opResolver, passResolver, section, logger)

	envVars, err := ope.GetEnvVars(ctx, itemRef)
	if err != nil {
		logger.ErrorContext(ctx, "failed to retrieve environment variables", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	outputEnvVars(envVars)

	return nil
}

var ErrNoAccountName = errors.New(
	"1Password account name not set, use --1password-account-name flag " +
		"or 1PASSWORD_ACCOUNT_NAME env var",
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/na4ma4/1password-direnv-tool/internal/openv"
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
	_ = viper.BindEnv("timeout", "OP_TIMEOUT")
}

func mainCmd(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	ctx, cancel := context.WithTimeout(ctx, viper.GetDuration("timeout"))
	defer cancel()

	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	logger.DebugContext(ctx, "enabled debug logging")

	if codec.Default == nil {
		logger.ErrorContext(ctx, "no codec available for decrypting item reference")
		return fmt.Errorf("%w%v", cmdconst.ErrNoUsage, "no codec available for decrypting item reference")
	}

	var cst cache.Cache
	{
		if viper.GetBool("cache.enabled") {
			cachePath := viper.GetString("cache.path")
			{
				var err error
				cst, err = cache.NewDisk(cachePath)
				if err != nil {
					logger.ErrorContext(ctx, "failed to initialize file cache", slogtool.ErrorAttr(err))
					return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
				}
			}
			if err := cst.Iterate(ctx, cache.ExpireFunc(viper.GetDuration("cache.age"))); err != nil {
				logger.ErrorContext(ctx, "failed to expire old cache entries", slogtool.ErrorAttr(err))
				return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
			}
			logger.DebugContext(ctx, "initialized file cache",
				slog.String("cache_path", cachePath),
				slog.Duration("cache_age", viper.GetDuration("cache.age")),
			)

			cst = cache.NewEncryption(cst, codec.Default)
		} else {
			cst = cache.NewNoop()
			logger.DebugContext(ctx, "caching disabled")
		}
	}

	var itemRef itemref.Ref
	{
		var err error
		itemRef, err = itemref.GetRef(ctx, codec.Default)
		if err != nil || itemRef.IsEmpty() {
			logger.ErrorContext(ctx, "failed to get item reference from configuration", slogtool.ErrorAttr(err))
			return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
		}
	}

	logger.InfoContext(ctx, "loading environment variables from 1Password", slog.String("item", itemRef.String()))

	lazyClient := cache.OnePasswordClientLazyInit(ctx, logger)
	section := viper.GetString("section")
	ope := openv.New(lazyClient, cst, section, logger)

	envVars, err := ope.GetEnvVars(ctx, itemRef)
	if err != nil {
		logger.ErrorContext(ctx, "failed to retrieve environment variables", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	for env := range envVars {
		// We are intentionally writing to stdout in a format that can be
		// eval'd by the caller, so we need to allow this.
		//nolint:gosec // intentional, see above.
		fmt.Fprintf(os.Stdout, "export %s=%s\n", env.Name, shellQuote(env.Value))
	}

	return nil
}

// ErrNoAccountName is returned when no 1Password account name is configured.
var ErrNoAccountName = errors.New(
	"1Password account name not set, use --1password-account-name flag " +
		"or 1PASSWORD_ACCOUNT_NAME env var",
)

// shellQuote wraps a string in single quotes, escaping any existing single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
)

// cleanCmd represents the base command when called without any subcommands.
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the cached values",
	Long:  `Clean the cached values. This is useful if you want to clear the cache after a password change or if you want to force a refresh of the cached values.`,
	RunE:  cleanCommand,
	Args:  cobra.NoArgs,
}

func cleanCommand(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	logger.Debug("Cleaning cached values")

	var cst cache.Cache
	{
		if viper.GetBool("cache.enabled") {
			cachePath := viper.GetString("cache.path")
			{
				var err error
				cst, err = cache.NewDisk(cachePath)
				if err != nil {
					logger.ErrorContext(ctx, "Failed to initialize file cache", slogtool.ErrorAttr(err))
					return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
				}
			}
			if err := cst.Iterate(ctx, cache.ExpireFunc(viper.GetDuration("cache.age"))); err != nil {
				logger.ErrorContext(ctx, "Failed to expire old cache entries", slogtool.ErrorAttr(err))
				return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
			}
			logger.DebugContext(ctx, "Initialized file cache",
				slog.String("cache_path", cachePath),
				slog.Duration("cache_age", viper.GetDuration("cache.age")),
			)
		} else {
			cst = cache.NewNoop()
			logger.DebugContext(ctx, "Caching disabled")
		}
	}

	if err := cst.Clear(ctx); err != nil {
		logger.ErrorContext(ctx, "Failed to clear cache", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	logger.Info("Cache cleared successfully")
	return nil
}

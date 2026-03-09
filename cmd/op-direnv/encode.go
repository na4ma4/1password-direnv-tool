package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
	"github.com/na4ma4/1password-direnv-tool/internal/codec"
)

// encodeCmd represents the base command when called without any subcommands.
var encodeCmd = &cobra.Command{
	Use:   "encode <item-ref>",
	Short: "Encode a 1Password item reference for use in environment variables",
	Long:  `Encode a 1Password item reference for use in environment variables. The item reference should be in the format "op://vault-name-or-id/item-name-or-id".`,
	RunE:  encodeCommand,
	Args:  cobra.MinimumNArgs(1),
}

func encodeCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	logger.Debug("encrypting item reference")

	encRef, err := codec.Default.Encode(args[0])
	if err != nil {
		logger.ErrorContext(ctx, "failed to encrypt item reference", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	fmt.Fprintln(os.Stdout, "encv1://"+encRef)
	return nil
}

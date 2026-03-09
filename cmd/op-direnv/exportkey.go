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

// exportCmd represents the command for exporting the encryption key used for encoding item references.
var exportKeyCmd = &cobra.Command{
	Use:   "export-key",
	Short: "Export the encryption key used for encoding item references",
	Long:  `Export the encryption key used for encoding item references. This key can be used to decrypt item references encoded with this tool.`,
	RunE:  exportKeyCommand,
	Args:  cobra.NoArgs,
}

func exportKeyCommand(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	logger.Debug("exporting encryption key")

	key, err := codec.Default.ExportKey()
	if err != nil {
		logger.ErrorContext(ctx, "failed to export encryption key", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	fmt.Fprintln(os.Stdout, "encv1-key://"+key)
	return nil
}

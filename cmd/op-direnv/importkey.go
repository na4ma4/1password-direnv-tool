package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"

	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
	"github.com/na4ma4/1password-direnv-tool/internal/codec"
)

// importKeyCmd represents the command for importing the encryption key used for encoding item references.
var importKeyCmd = &cobra.Command{
	Use:   "import-key",
	Short: "Import the encryption key used for encoding item references",
	Long:  `Import the encryption key used for encoding item references. This key can be used to decrypt item references encoded with this tool.`,
	RunE:  importKeyCommand,
	Args:  cobra.NoArgs,
}

func importKeyCommand(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	logger.Debug("Importing encryption key")

	fmt.Fprint(os.Stdout, "Enter the encryption key: ")
	reader := bufio.NewReader(os.Stdin)
	var input string
	{
		var err error
		input, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
		}
	}

	if err := codec.Default.ImportKey(input); err != nil {
		logger.ErrorContext(ctx, "Failed to import encryption key", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	fmt.Fprintln(os.Stdout, "Encryption key imported successfully")
	return nil
}

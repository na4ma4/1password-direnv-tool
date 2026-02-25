package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	"github.com/dosquad/go-cliversion"
	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
	"github.com/na4ma4/1password-direnv-tool/internal/openv"
	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:          "op-direnv",
	Short:        "1Password direnv tool",
	Long:         `1Password direnv tool loads environment variables from 1Password for use with direnv.`,
	RunE:         mainCmd,
	Args:         cobra.NoArgs,
	Version:      cliversion.Get().VersionString(),
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug output")
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindEnv("debug", "DEBUG")

	rootCmd.PersistentFlags().StringP("item", "i", "", "1Password item reference (e.g. op://vault/item)")
	_ = viper.BindPFlag("item", rootCmd.PersistentFlags().Lookup("item"))
	_ = viper.BindEnv("item", "OP_ITEM_UUID")

	rootCmd.PersistentFlags().StringP("service-account-token", "t", "", "1Password service account token")
	_ = viper.BindPFlag("service-account-token", rootCmd.PersistentFlags().Lookup("service-account-token"))
	_ = viper.BindEnv("service-account-token", "OP_SERVICE_ACCOUNT_TOKEN")

	rootCmd.PersistentFlags().StringP("section", "s", "Environment", "Section name containing environment variables")
	_ = viper.BindPFlag("section", rootCmd.PersistentFlags().Lookup("section"))
	_ = viper.BindEnv("section", "OP_SECTION")
}

func main() {
	if err := rootCmd.Execute(); errors.Is(err, cmdconst.ErrNoUsage) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	} else if err != nil {
		_ = rootCmd.Usage()
		os.Exit(1)
	}
}

func mainCmd(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	logger.Debug("Debug logging enabled")

	itemRef := viper.GetString("item")
	if itemRef == "" {
		var err error
		itemRef, err = getGitConfig("1password.env-item")

		if err != nil || itemRef == "" {
			return fmt.Errorf(
				"%witem reference not set, use --item flag, OP_ITEM_UUID env var, or git config 1password.env-item",
				cmdconst.ErrNoUsage,
			)
		}
	}

	logger.InfoContext(ctx, "Loading environment variables from 1Password", slog.String("item", itemRef))

	client, err := createClient(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create 1Password client", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	section := viper.GetString("section")

	envVars, err := openv.GetEnvVars(ctx, client, itemRef, section, logger)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve environment variables", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	for _, env := range envVars {
		fmt.Printf("export %s=%s\n", env.Name, shellQuote(env.Value))
	}

	return nil
}

// ErrNoToken is returned when no service account token is configured.
var ErrNoToken = errors.New("service account token not set, use --service-account-token flag or OP_SERVICE_ACCOUNT_TOKEN env var")

func createClient(ctx context.Context) (*onepassword.Client, error) {
	token := viper.GetString("service-account-token")

	if token == "" {
		return nil, ErrNoToken
	}

	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("1Password direnv tool", cliversion.Get().VersionString()),
	)
	if err != nil {
		return nil, fmt.Errorf("creating 1Password client: %w", err)
	}

	return client, nil
}

// getGitConfig retrieves a git config value by key.
func getGitConfig(key string) (string, error) {
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		return "", fmt.Errorf("git config --get %q: %w", key, err)
	}

	return strings.TrimSpace(string(out)), nil
}

// shellQuote wraps a string in single quotes, escaping any existing single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

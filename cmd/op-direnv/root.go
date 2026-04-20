package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dosquad/go-cliversion"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
)

const (
	cacheAgeDefault = 8 * time.Hour
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

	rootCmd.PersistentFlags().StringP("1password-account-name", "a", "",
		"1Password account name for desktop app integration")
	_ = viper.BindPFlag("1password.account-name", rootCmd.PersistentFlags().Lookup("1password-account-name"))
	_ = viper.BindEnv("1password.account-name", "1PASSWORD_ACCOUNT_NAME")

	rootCmd.PersistentFlags().StringP("item", "i", "",
		"1Password item reference (e.g. op://vault/item)")
	_ = viper.BindPFlag("item", rootCmd.PersistentFlags().Lookup("item"))
	_ = viper.BindEnv("item", "OP_ITEM_UUID")

	rootCmd.PersistentFlags().BoolP("cache", "c", true,
		"Enable caching of decrypted item references")
	_ = viper.BindPFlag("cache.enabled", rootCmd.PersistentFlags().Lookup("cache"))
	_ = viper.BindEnv("cache.enabled", "OP_CACHE_ENABLED")

	rootCmd.PersistentFlags().Duration("cache-age", cacheAgeDefault,
		"Duration for which decrypted item references are cached")
	_ = viper.BindPFlag("cache.age", rootCmd.PersistentFlags().Lookup("cache-age"))
	_ = viper.BindEnv("cache.age", "OP_CACHE_AGE")

	rootCmd.PersistentFlags().Bool("lock", true,
		"Serialize concurrent op-direnv invocations with a file lock")
	_ = viper.BindPFlag("lock.enabled", rootCmd.PersistentFlags().Lookup("lock"))
	_ = viper.BindEnv("lock.enabled", "OP_LOCK")

	rootCmd.PersistentFlags().Duration("lock-timeout", 0,
		"Max time to wait for the file lock (0 = use --timeout)")
	_ = viper.BindPFlag("lock.timeout", rootCmd.PersistentFlags().Lookup("lock-timeout"))
	_ = viper.BindEnv("lock.timeout", "OP_LOCK_TIMEOUT")

	rootCmd.AddCommand(exportKeyCmd)
	rootCmd.AddCommand(encodeCmd)
	rootCmd.AddCommand(importKeyCmd)
	rootCmd.AddCommand(cleanCmd)
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

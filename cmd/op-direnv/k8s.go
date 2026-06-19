package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/na4ma4/go-slogtool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
	"github.com/na4ma4/1password-direnv-tool/internal/codec"
	"github.com/na4ma4/1password-direnv-tool/internal/itemref"
	"github.com/na4ma4/1password-direnv-tool/internal/kubecfg"
	"github.com/na4ma4/1password-direnv-tool/model"
)

// k8sCmd represents the command for use as a exec-plugin for kubernetes.
var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Run as a kubernetes exec-plugin to provide an ExecCredential for kubectl authentication",
	Long:  `Run as a kubernetes exec-plugin to provide an ExecCredential for kubectl authentication.`,
	RunE:  k8sCommand,
	Args:  cobra.MaximumNArgs(1),
}

func k8sCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	logLevel := slog.LevelInfo
	if viper.GetBool("debug") {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	logger.DebugContext(ctx, "running in kubernetes exec-plugin mode")

	var cst cache.Cache
	{
		var err error
		cst, err = setupCache(ctx, logger)
		if err != nil {
			return err
		}
	}

	var itemRef itemref.Ref
	{
		var err error
		itemRef, err = itemref.GetRef(ctx, codec.Default, args)
		if err != nil || itemRef.IsEmpty() {
			logger.ErrorContext(ctx, "failed to get item reference from configuration", slogtool.ErrorAttr(err))
			return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
		}
	}

	var refURL *url.URL
	{
		var err error
		refURL, err = url.Parse(itemRef.String())
		if err != nil {
			logger.ErrorContext(ctx, "failed to parse item reference URI",
				slog.String("ref", itemRef.String()),
				slogtool.ErrorAttr(err),
			)
			return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
		}
	}

	scheme := refURL.Scheme

	var p model.Provider
	{
		var err error
		p, err = getProvider(ctx, logger, scheme, itemRef, cst)
		if err != nil {
			return err
		}
	}

	t := kubecfg.NewProvider(p)

	var cred *model.ExecCredential
	{
		var err error
		cred, err = t.LookupExecCredential(ctx, itemRef.String())
		if err != nil {
			logger.ErrorContext(ctx, "failed to lookup ExecCredential", slogtool.ErrorAttr(err))
			return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(cred); err != nil {
		logger.ErrorContext(ctx, "failed to encode ExecCredential to JSON", slogtool.ErrorAttr(err))
		return fmt.Errorf("%w%w", cmdconst.ErrNoUsage, err)
	}

	return nil
}

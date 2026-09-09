package main

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/na4ma4/1password-direnv-tool/internal/cmdconst"
	"github.com/na4ma4/1password-direnv-tool/internal/codec"
	"github.com/na4ma4/1password-direnv-tool/model"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// cacheCmd represents the base command when called without any subcommands.
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the cached values",
	Long:  `Manage the cached values. This includes cleaning the cache and potentially other cache-related operations in the future.`,
}

var cacheListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List cached values",
	Long:    `List all the values currently stored in the cache.`,
	RunE:    cacheListCommand,
}

var cacheGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a cached value",
	Long:  `Retrieve a value from the cache by its key.`,
	RunE:  cacheGetCommand,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the cached values",
	Long:  `Clean the cached values. This is useful if you want to clear the cache after a password change or if you want to force a refresh of the cached values.`,
	RunE:  cleanCommand,
}

func init() {
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheGetCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
}

func cacheListCommand(cmd *cobra.Command, args []string) error {
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

	outChan := make(chan *cacheEntry)

	go func() {
		_ = cst.Iterate(ctx, func(key string, files *model.FileList, age time.Time, value string) error {
			outChan <- &cacheEntry{
				Key:   key,
				Files: files.ToSlice(),
				Age:   age,
				Value: value,
			}

			return nil
		})
		close(outChan)
	}()

	return printCacheEnties(outChan, defaultCacheTemplate)
}

type cacheEntry struct {
	Key   string
	Files []string
	Age   time.Time
	Value string
}

func (cacheEntry) Headers() map[string]string {
	return map[string]string{
		"Key":   "Key",
		"Files": "Files",
		"Age":   "Age",
		"Value": "Value",
	}
}

const (
	defaultCacheTemplate = "{{.Key}}\t{{.Files}}\t{{if typeIs \"time.Time\" .Age}}{{ago .Age}}{{else}}{{.Age}}{{end}}\n"
)

func printCacheEnties(ch <-chan *cacheEntry, tmplStr string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	if !strings.HasSuffix(tmplStr, "\n") {
		tmplStr += "\n"
	}

	var tmpl *template.Template
	{
		var err error
		tmpl, err = template.New("cache").Funcs(sprig.FuncMap()).Parse(tmplStr)
		if err != nil {
			return err
		}
	}

	var once sync.Once

	for entry := range ch {
		once.Do(func() {
			if err := tmpl.Execute(w, entry.Headers()); err != nil {
				return
			}
		})

		if err := tmpl.Execute(w, entry); err != nil {
			return err
		}
	}

	return nil
}

func cacheGetCommand(cmd *cobra.Command, args []string) error {
	// Implement the cache get command logic here
	return nil
}

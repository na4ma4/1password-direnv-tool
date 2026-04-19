package modifier

import (
	"log/slog"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
	"github.com/na4ma4/1password-direnv-tool/internal/opclient"
)

type regoptions struct {
	logger    *slog.Logger
	modifiers []Modifier
}

type regoptionsFunc func(*regoptions)

func (o *regoptions) applyDefaults() {
	if o.logger == nil {
		o.logger = slog.New(slog.DiscardHandler)
	}
}

// WithRegistryLogger sets the logger to be used by the registry.
//
//nolint:revive // WithRegistryLogger is not intended to be used as an interface
func WithRegistryLogger(logger *slog.Logger) regoptionsFunc {
	return func(o *regoptions) {
		o.logger = logger
	}
}

// WithModifiers adds modifiers to the registry.
//
//nolint:revive // WithModifiers is not intended to be used as an interface
func WithModifiers(modifiers ...Modifier) regoptionsFunc {
	return func(o *regoptions) {
		o.modifiers = append(o.modifiers, modifiers...)
	}
}

type options struct {
	client opclient.Func
	logger *slog.Logger
	cache  cache.Cache
}

type optionsFunc func(*options)

func (o *options) applyDefaults() {
	if o.logger == nil {
		o.logger = slog.New(slog.DiscardHandler)
	}
}

// WithLogger sets the logger to be used by the generator.
//
//nolint:revive // WithLogger is not intended to be used as an interface
func WithLogger(logger *slog.Logger) optionsFunc {
	return func(o *options) {
		o.logger = logger
	}
}

// WithOnePasswordClient sets the 1Password client to be used by the generator.
//
//nolint:revive // WithOnePasswordClient is not intended to be used as an interface
func WithOnePasswordClient(client opclient.Func) optionsFunc {
	return func(o *options) {
		o.client = client
	}
}

// WithCache sets the cache to be used by the generator.
//
//nolint:revive // WithCache is not intended to be used as an interface
func WithCache(cache cache.Cache) optionsFunc {
	return func(o *options) {
		o.cache = cache
	}
}

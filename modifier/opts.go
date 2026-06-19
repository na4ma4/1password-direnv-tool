package modifier

import (
	"log/slog"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type regoptions struct {
	logger    *slog.Logger
	modifiers []model.Modifier
}

type regoptionsFunc func(*regoptions)

func (o *regoptions) applyDefaults() {
	if o.logger == nil {
		o.logger = slog.New(slog.DiscardHandler)
	}
}

// WithRegistryLogger sets the logger for the modifier registry. If not set, a no-op logger will be used.
//
//nolint:revive // we want to keep the "With" prefix for consistency with other providers
func WithRegistryLogger(logger *slog.Logger) regoptionsFunc {
	return func(o *regoptions) {
		o.logger = logger
	}
}

// WithModifiers registers the given modifiers with the registry.
//
//nolint:revive // we want to keep the "With" prefix for consistency with other providers
func WithModifiers(modifiers ...model.Modifier) regoptionsFunc {
	return func(o *regoptions) {
		o.modifiers = append(o.modifiers, modifiers...)
	}
}

type options struct {
	resolver model.SecretResolver
	logger   *slog.Logger
}

type optionsFunc func(*options)

func (o *options) applyDefaults() {
	if o.logger == nil {
		o.logger = slog.New(slog.DiscardHandler)
	}
}

// WithLogger sets the logger for the modifier. If not set, a no-op logger will be used.
//
//nolint:revive // we want to keep the "With" prefix for consistency with other providers
func WithLogger(logger *slog.Logger) optionsFunc {
	return func(o *options) {
		o.logger = logger
	}
}

// WithSecretResolver sets the secret resolver for the modifier. If not set, the modifier will not be able to resolve secrets.
//
//nolint:revive // we want to keep the "With" prefix for consistency with other providers
func WithSecretResolver(resolver model.SecretResolver) optionsFunc {
	return func(o *options) {
		o.resolver = resolver
	}
}

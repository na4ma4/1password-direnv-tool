package modifier

import (
	"context"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
)

type OnePasswordModifier struct {
	opts *options
}

func NewOnePassword(opts ...optionsFunc) *OnePasswordModifier {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}
	options.applyDefaults()

	if options.client == nil {
		panic("OnePassword client is required for OnePasswordModifier")
	}

	return &OnePasswordModifier{opts: options}
}

func (m *OnePasswordModifier) Tags() Tags {
	return Tags{"1password"}
}

func (m *OnePasswordModifier) opSecretResolve(ctx context.Context, secretRef string) (string, error) {
	return cache.OnePasswordSecretResolve(ctx, m.opts.cache, m.opts.logger, m.opts.client, secretRef)
}

func (m *OnePasswordModifier) Apply(ctx context.Context, value string) (string, error) {
	// The value is expected to be in the format "op://vault/item/field"
	resolved, err := m.opSecretResolve(ctx, value)
	if err != nil {
		return "", err
	}

	return resolved, nil
}

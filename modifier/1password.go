package modifier

import (
	"context"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type OnePasswordModifier struct {
	resolver model.SecretResolver
	opts     *options
}

func NewOnePassword(resolver model.SecretResolver, opts ...optionsFunc) *OnePasswordModifier {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}
	options.applyDefaults()

	return &OnePasswordModifier{resolver: resolver, opts: options}
}

func (m *OnePasswordModifier) Tags() model.Tags {
	return model.Tags{"1password", "op"}
}

func (m *OnePasswordModifier) Apply(ctx context.Context, value *model.SecretRef) error {
	err := m.resolver.ResolveSecret(ctx, value)
	if err != nil {
		return err
	}

	return nil
}

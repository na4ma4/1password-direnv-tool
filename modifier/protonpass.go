package modifier

import (
	"context"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type ProtonPassModifier struct {
	resolver model.SecretResolver
	opts     *options
}

func NewProtonPass(resolver model.SecretResolver, opts ...optionsFunc) *ProtonPassModifier {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}
	options.applyDefaults()

	return &ProtonPassModifier{resolver: resolver, opts: options}
}

func (m *ProtonPassModifier) Tags() model.Tags {
	return model.Tags{"protonpass", "pass"}
}

func (m *ProtonPassModifier) Apply(ctx context.Context, value *model.SecretRef) error {
	err := m.resolver.ResolveSecret(ctx, value)
	if err != nil {
		return err
	}

	return nil
}

package modifier

import (
	"context"
	"encoding/base32"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type Base32Modifier struct {
	opts *options
}

func NewBase32(opts ...optionsFunc) *Base32Modifier {
	options := &options{}

	for _, opt := range opts {
		opt(options)
	}

	options.applyDefaults()

	return &Base32Modifier{opts: options}
}

func (m *Base32Modifier) Tags() model.Tags {
	return model.Tags{"b32", "base32"}
}

func (m *Base32Modifier) Apply(_ context.Context, value string) (string, error) {
	encoded := base32.StdEncoding.EncodeToString([]byte(value))
	return encoded, nil
}

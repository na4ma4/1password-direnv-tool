package modifier

import (
	"context"
	"encoding/base32"
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

func (m *Base32Modifier) Tags() Tags {
	return Tags{"b32", "base32"}
}

func (m *Base32Modifier) Apply(_ context.Context, value string) (string, error) {
	encoded := base32.StdEncoding.EncodeToString([]byte(value))
	return encoded, nil
}

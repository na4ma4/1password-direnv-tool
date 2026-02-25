package modifier

import (
	"context"
	"encoding/base64"
)

type Base64Modifier struct {
	opts *options
}

func NewBase64(opts ...optionsFunc) *Base64Modifier {
	options := &options{}

	for _, opt := range opts {
		opt(options)
	}

	options.applyDefaults()

	return &Base64Modifier{opts: options}
}

func (m *Base64Modifier) Tags() Tags {
	return Tags{"b64", "base64"}
}

func (m *Base64Modifier) Apply(_ context.Context, value string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(value))
	return encoded, nil
}

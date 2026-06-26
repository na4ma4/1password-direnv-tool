package modifier

import (
	"log/slog"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type registry struct {
	logger    *slog.Logger
	modifiers []model.Modifier
}

// NewRegistry creates a new registry with the provided options.
//
//nolint:revive // registry is not intended to be used as an interface
func NewRegistry(opts ...regoptionsFunc) *registry {
	options := &regoptions{}
	for _, opt := range opts {
		opt(options)
	}
	options.applyDefaults()

	return &registry{modifiers: options.modifiers, logger: options.logger}
}

func (r *registry) Add(m model.Modifier) *registry {
	r.modifiers = append(r.modifiers, m)

	return r
}

func (r *registry) GetModifiers() []model.Modifier {
	return r.modifiers
}

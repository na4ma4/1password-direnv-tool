package modifier

import (
	"context"
	"fmt"
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

func (r *registry) Apply(ctx context.Context, value string, modifierNames []string) (string, error) {
	for _, name := range modifierNames {
		var found bool
		for _, m := range r.modifiers {
			if m.Tags().Contains(name) {
				r.logger.DebugContext(ctx, "applying modifier",
					slog.String("tag", name),
				)
				var err error
				value, err = m.Apply(ctx, value)
				if err != nil {
					return "", fmt.Errorf("applying modifier %q: %w", name, err)
				}
				found = true
				break
			}
		}

		if !found {
			return "", fmt.Errorf("unknown modifier: %s", name)
		}
	}

	return value, nil
}

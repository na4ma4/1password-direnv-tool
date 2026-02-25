package modifier

import (
	"context"
	"slices"
)

type Modifier interface {
	Tags() Tags
	Apply(ctx context.Context, value string) (string, error)
}

type Tags []string

func (t Tags) Contains(tag string) bool {
	return slices.Contains(t, tag)
}

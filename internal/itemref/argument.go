package itemref

import (
	"context"
	"errors"
)

type argument struct {
	value string
}

func (a *argument) Version() RefVersion {
	return coreRefType(a)
}

func (a *argument) IsEmpty() bool {
	return coreIsEmpty(a)
}

func (a *argument) String() string {
	return a.value
}

// getArgumentConfig retrieves a git config value by key.
func getArgumentConfig(_ context.Context, args []string) (*argument, error) {
	if len(args) >= 1 {
		return &argument{value: args[0]}, nil
	}

	return nil, errors.New("no arguments provided")
}

package itemref

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type git struct {
	value string
}

func (g *git) Version() RefVersion {
	return coreRefType(g)
}

func (g *git) IsEmpty() bool {
	return coreIsEmpty(g)
}

func (g *git) String() string {
	return g.value
}

// getGitConfig retrieves a git config value by key.
func getGitConfig(ctx context.Context, key string) (*git, error) {
	out, err := exec.CommandContext(ctx, "git", "config", "--get", key).Output()
	if err != nil {
		return nil, fmt.Errorf("git config --get %q: %w", key, err)
	}

	return &git{value: strings.TrimSpace(string(out))}, nil
}

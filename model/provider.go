package model

import (
	"context"
)

type SecretResolver interface {
	ResolveSecret(ctx context.Context, ref string) (string, error)
	RetrieveK8sCredential(ctx context.Context, ref string) (*ExecCredential, error)
}

type EnvItem struct {
	Name      string
	Value     string
	Modifiers []string
}

type Provider interface {
	Name() string

	LookupEnvVars(ctx context.Context, ref string, section string) ([]EnvItem, error)

	SecretResolver() SecretResolver
}

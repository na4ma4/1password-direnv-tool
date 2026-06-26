package model

import (
	"context"
)

type SecretResolver interface {
	ResolveSecret(ctx context.Context, ref *SecretRef) error
	RetrieveK8sCredential(ctx context.Context, ref string) (*ExecCredential, *FileList, error)
}

type Provider interface {
	Name() string

	LookupEnvVars(ctx context.Context, ref string, section string) ([]EnvVar, error)

	SecretResolver() SecretResolver
}

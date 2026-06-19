package lazy

import (
	"context"
	"sync"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type Resolver struct {
	factory  func() (model.SecretResolver, error)
	once     sync.Once
	resolver model.SecretResolver
	err      error
}

func NewResolver(factory func() (model.SecretResolver, error)) *Resolver {
	return &Resolver{factory: factory}
}

func (r *Resolver) init() {
	r.once.Do(func() {
		r.resolver, r.err = r.factory()
	})
}

func (r *Resolver) ResolveSecret(ctx context.Context, ref string) (string, error) {
	r.init()
	if r.err != nil {
		return "", r.err
	}
	return r.resolver.ResolveSecret(ctx, ref)
}

func (r *Resolver) RetrieveK8sCredential(ctx context.Context, ref string) (*model.ExecCredential, error) {
	r.init()
	if r.err != nil {
		return nil, r.err
	}
	return r.resolver.RetrieveK8sCredential(ctx, ref)
}

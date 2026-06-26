package kubecfg

import (
	"context"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type Provider struct {
	p model.Provider
}

func NewProvider(p model.Provider) *Provider {
	return &Provider{p: p}
}

func (p *Provider) Name() string {
	return p.p.Name()
}

func (p *Provider) LookupExecCredential(
	ctx context.Context,
	ref string,
) (*model.ExecCredential, *model.FileList, error) {
	secret, files, err := p.p.SecretResolver().RetrieveK8sCredential(ctx, ref)
	if err != nil {
		return nil, nil, err
	}

	return secret, files, nil
}

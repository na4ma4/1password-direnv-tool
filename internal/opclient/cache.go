package opclient

// import (
// 	"context"

// 	"github.com/1password/onepassword-sdk-go"
// 	"github.com/na4ma4/1password-direnv-tool/internal/cache"
// )

// type CachingClient struct {
// 	c  *onepassword.Client
// 	cc cache.Cache
// }

// func NewCachingClient(client *onepassword.Client, cache cache.Cache) *CachingClient {
// 	return &CachingClient{
// 		c:  client,
// 		cc: cache,
// 	}
// }

// func (cc *CachingClient) Secrets() onepassword.SecretsAPI {
// 	return cc.c.Secrets()
// }

// type CachingSecretsAPI struct {
// 	c     *onepassword.Client
// 	cache cache.Cache
// }

// // Resolve returns the secret the provided secret reference points to.
// func (cs *CachingSecretsAPI) Resolve(ctx context.Context, secretReference string) (string, error) {
// 	item, _, err := cs.cache.Get(ctx, "secrets:"+secretReference)
// 	if err != nil {
// 		return cs.c.Secrets().Resolve(ctx, secretReference)
// 	}

// 	return item, nil
// }

// // Resolve takes in a list of secret references and returns the secrets they point to or errors if any.
// func (cs *CachingSecretsAPI) ResolveAll(ctx context.Context, secretReferences []string) (onepassword.ResolveAllResponse, error) {

// }

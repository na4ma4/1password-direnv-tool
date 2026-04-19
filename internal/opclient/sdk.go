package opclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/dosquad/go-cliversion"
)

func NewSDKClient(ctx context.Context, accountName string) (Client, error) {
	resultCh := make(chan newClientResult, 1)

	go func() {
		defer close(resultCh)
		client, err := onepassword.NewClient(
			ctx,
			onepassword.WithDesktopAppIntegration(accountName),
			onepassword.WithIntegrationInfo("1Password direnv tool", cliversion.Get().VersionString()),
		)

		resultCh <- newClientResult{client: client, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("creating 1Password client canceled: %w", ctx.Err())
	case result, ok := <-resultCh:
		if !ok {
			return nil, errors.New("creating 1Password client: result channel closed unexpectedly")
		}

		if result.err != nil {
			return nil, fmt.Errorf("creating 1Password client: %w", result.err)
		}

		return result.client, nil
	}
}

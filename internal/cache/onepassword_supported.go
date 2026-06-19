//go:build cgo || windows

package cache

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/dosquad/go-cliversion"
)

func onepasswordNewClient(ctx context.Context, accountName string) (*onepassword.Client, error) {
	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithDesktopAppIntegration(accountName),
		onepassword.WithIntegrationInfo("1Password direnv tool", cliversion.Get().VersionString()),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}

package opclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/1password/onepassword-sdk-go"
)

// CLIClient implements the Client interface using the 1Password CLI (op) tool.
type CLIClient struct {
	accountName string
}

// NewCLIClient creates a new CLI-based 1Password client.
func NewCLIClient(_ context.Context, accountName string) (Client, error) {
	if accountName == "" {
		accountName = "my"
	}

	// Verify op is available
	if _, err := exec.LookPath("op"); err != nil {
		return nil, fmt.Errorf("op CLI not found: %w", err)
	}

	return &CLIClient{accountName: accountName}, nil
}

func (c *CLIClient) Secrets() onepassword.SecretsAPI {
	return &cliSecretsAPI{client: c}
}

func (c *CLIClient) Items() onepassword.ItemsAPI {
	return &cliItemsAPI{client: c}
}

func (c *CLIClient) Vaults() onepassword.VaultsAPI {
	return &cliVaultsAPI{client: c}
}

func (c *CLIClient) Groups() onepassword.GroupsAPI {
	return &cliGroupsAPI{}
}

// runOP executes the op CLI with the given arguments.
func (c *CLIClient) runOP(ctx context.Context, args ...string) (string, error) {
	fullArgs := make([]string, 0, len(args)+2) //nolint:mnd // +2 for --account and account name
	copy(fullArgs, []string{"--account", c.accountName})
	fullArgs = append(fullArgs, args...)

	cmd := exec.CommandContext(ctx, "op", fullArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("op %s failed: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.String(), nil
}

// cliSecretsAPI implements SecretsAPI using the op CLI.
type cliSecretsAPI struct {
	client *CLIClient
}

func (s *cliSecretsAPI) Resolve(ctx context.Context, secretReference string) (string, error) {
	output, err := s.client.runOP(ctx, "read", secretReference, "--no-newline")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (s *cliSecretsAPI) ResolveAll(_ context.Context, _ []string) (onepassword.ResolveAllResponse, error) {
	return onepassword.ResolveAllResponse{}, errors.New("ResolveAll not supported by CLI client")
}

// cliItemsAPI implements ItemsAPI using the op CLI.
type cliItemsAPI struct {
	client *CLIClient
}

func (i *cliItemsAPI) Get(ctx context.Context, vaultID, itemID string) (onepassword.Item, error) {
	var output string
	{
		var err error
		output, err = i.client.runOP(ctx, "item", "get", itemID, "--vault", vaultID, "--format", "json")
		if err != nil {
			return onepassword.Item{}, err
		}
	}

	var item onepassword.Item
	if err := json.NewDecoder(strings.NewReader(output)).Decode(&item); err != nil {
		return onepassword.Item{}, fmt.Errorf("failed to decode item response: %w", err)
	}

	return item, nil
}

func (i *cliItemsAPI) List(
	ctx context.Context,
	vaultID string,
	_ ...onepassword.ItemListFilter,
) ([]onepassword.ItemOverview, error) {
	var output string
	{
		var err error
		output, err = i.client.runOP(ctx, "item", "list", "--vault", vaultID, "--format", "json")
		if err != nil {
			return nil, err
		}
	}

	var items []onepassword.ItemOverview
	if err := json.NewDecoder(strings.NewReader(output)).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode item list response: %w", err)
	}

	return items, nil
}

func (i *cliItemsAPI) Create(_ context.Context, _ onepassword.ItemCreateParams) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("Create not supported by CLI client")
}

func (i *cliItemsAPI) CreateAll(
	_ context.Context,
	_ string,
	_ []onepassword.ItemCreateParams,
) (onepassword.ItemsUpdateAllResponse, error) {
	return onepassword.ItemsUpdateAllResponse{}, errors.New("CreateAll not supported by CLI client")
}

func (i *cliItemsAPI) GetAll(_ context.Context, _ string, _ []string) (onepassword.ItemsGetAllResponse, error) {
	return onepassword.ItemsGetAllResponse{}, errors.New("GetAll not supported by CLI client")
}

func (i *cliItemsAPI) Put(_ context.Context, _ onepassword.Item) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("Put not supported by CLI client")
}

func (i *cliItemsAPI) Delete(_ context.Context, _, _ string) error {
	return errors.New("Delete not supported by CLI client")
}

func (i *cliItemsAPI) DeleteAll(_ context.Context, _ string, _ []string) (onepassword.ItemsDeleteAllResponse, error) {
	return onepassword.ItemsDeleteAllResponse{}, errors.New("DeleteAll not supported by CLI client")
}

func (i *cliItemsAPI) Archive(_ context.Context, _, _ string) error {
	return errors.New("Archive not supported by CLI client")
}

func (i *cliItemsAPI) Shares() onepassword.ItemsSharesAPI {
	return &cliItemsSharesAPI{}
}

func (i *cliItemsAPI) Files() onepassword.ItemsFilesAPI {
	return &cliItemsFilesAPI{}
}

// cliItemsSharesAPI implements ItemsSharesAPI with unsupported operations.
type cliItemsSharesAPI struct{}

func (c *cliItemsSharesAPI) GetAccountPolicy(
	_ context.Context,
	_, _ string,
) (onepassword.ItemShareAccountPolicy, error) {
	return onepassword.ItemShareAccountPolicy{}, errors.New("Shares.GetAccountPolicy not supported by CLI client")
}

func (c *cliItemsSharesAPI) ValidateRecipients(
	_ context.Context,
	_ onepassword.ItemShareAccountPolicy,
	_ []string,
) ([]onepassword.ValidRecipient, error) {
	return nil, errors.New("Shares.ValidateRecipients not supported by CLI client")
}

func (c *cliItemsSharesAPI) Create(
	_ context.Context,
	_ onepassword.Item,
	_ onepassword.ItemShareAccountPolicy,
	_ onepassword.ItemShareParams,
) (string, error) {
	return "", errors.New("Shares.Create not supported by CLI client")
}

// cliItemsFilesAPI implements ItemsFilesAPI with unsupported operations.
type cliItemsFilesAPI struct{}

func (c *cliItemsFilesAPI) Attach(
	_ context.Context,
	_ onepassword.Item,
	_ onepassword.FileCreateParams,
) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("Files.Attach not supported by CLI client")
}

func (c *cliItemsFilesAPI) Read(_ context.Context, _, _ string, _ onepassword.FileAttributes) ([]byte, error) {
	return nil, errors.New("Files.Read not supported by CLI client")
}

func (c *cliItemsFilesAPI) Delete(_ context.Context, _ onepassword.Item, _, _ string) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("Files.Delete not supported by CLI client")
}

func (c *cliItemsFilesAPI) ReplaceDocument(
	_ context.Context,
	_ onepassword.Item,
	_ onepassword.DocumentCreateParams,
) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("Files.ReplaceDocument not supported by CLI client")
}

// cliVaultsAPI implements VaultsAPI using the op CLI.
type cliVaultsAPI struct {
	client *CLIClient
}

func (v *cliVaultsAPI) List(
	ctx context.Context,
	_ ...onepassword.VaultListParams,
) ([]onepassword.VaultOverview, error) {
	var output string
	{
		var err error
		output, err = v.client.runOP(ctx, "vault", "list", "--format", "json")
		if err != nil {
			return nil, err
		}
	}

	var vaults []onepassword.VaultOverview
	if err := json.NewDecoder(strings.NewReader(output)).Decode(&vaults); err != nil {
		return nil, fmt.Errorf("failed to decode vault list response: %w", err)
	}

	return vaults, nil
}

func (v *cliVaultsAPI) GetOverview(_ context.Context, _ string) (onepassword.VaultOverview, error) {
	return onepassword.VaultOverview{}, errors.New("GetOverview not supported by CLI client")
}

func (v *cliVaultsAPI) Create(_ context.Context, _ onepassword.VaultCreateParams) (onepassword.Vault, error) {
	return onepassword.Vault{}, errors.New("Create not supported by CLI client")
}

func (v *cliVaultsAPI) Get(_ context.Context, _ string, _ onepassword.VaultGetParams) (onepassword.Vault, error) {
	return onepassword.Vault{}, errors.New("Get not supported by CLI client")
}

func (v *cliVaultsAPI) Update(_ context.Context, _ string, _ onepassword.VaultUpdateParams) (onepassword.Vault, error) {
	return onepassword.Vault{}, errors.New("Update not supported by CLI client")
}

func (v *cliVaultsAPI) Delete(_ context.Context, _ string) error {
	return errors.New("Delete not supported by CLI client")
}

func (v *cliVaultsAPI) GrantGroupPermissions(_ context.Context, _ string, _ []onepassword.GroupAccess) error {
	return errors.New("GrantGroupPermissions not supported by CLI client")
}

func (v *cliVaultsAPI) UpdateGroupPermissions(_ context.Context, _ []onepassword.GroupVaultAccess) error {
	return errors.New("UpdateGroupPermissions not supported by CLI client")
}

func (v *cliVaultsAPI) RevokeGroupPermissions(_ context.Context, _, _ string) error {
	return errors.New("RevokeGroupPermissions not supported by CLI client")
}

// cliGroupsAPI implements GroupsAPI with unsupported operations.
type cliGroupsAPI struct{}

func (g *cliGroupsAPI) Get(_ context.Context, _ string, _ onepassword.GroupGetParams) (onepassword.Group, error) {
	return onepassword.Group{}, errors.New("Get not supported by CLI client")
}

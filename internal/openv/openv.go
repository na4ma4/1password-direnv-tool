package openv

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	"github.com/na4ma4/go-slogtool"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
	"github.com/na4ma4/1password-direnv-tool/internal/itemref"
	"github.com/na4ma4/1password-direnv-tool/internal/modifier"
)

// validEnvVarName matches valid POSIX shell variable names: a letter or underscore
// followed by zero or more letters, digits, or underscores.
var validEnvVarName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Generator struct {
	client  cache.OPClientFunc
	cache   cache.Cache
	section string
	logger  *slog.Logger
}

func New(
	client cache.OPClientFunc,
	cache cache.Cache,
	section string,
	logger *slog.Logger,
) *Generator {
	return &Generator{
		client:  client,
		cache:   cache,
		section: section,
		logger:  logger,
	}
}

// GetEnvVars retrieves environment variables from a 1Password item.
// The itemRef should be in the format "op://vault-name-or-id/item-name-or-id".
// The section parameter specifies the section name to read from (e.g. "Environment").
func (g *Generator) GetEnvVars(ctx context.Context, itemRef itemref.Ref) (<-chan EnvVar, error) {
	vaultRef, itemName, err := g.parseItemRef(itemRef)
	if err != nil {
		return nil, fmt.Errorf("parsing item reference %q: %w", itemRef, err)
	}

	g.logger.DebugContext(ctx, "resolving vault", slog.String("vault", vaultRef))

	vaultID, err := g.resolveVaultID(ctx, vaultRef)
	if err != nil {
		return nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
	}

	g.logger.DebugContext(ctx, "resolving item", slog.String("item", itemName), slog.String("vault_id", vaultID))

	itemID, err := g.resolveItemID(ctx, vaultID, itemName)
	if err != nil {
		return nil, fmt.Errorf("resolving item %q: %w", itemName, err)
	}

	g.logger.DebugContext(ctx, "getting item", slog.String("item_id", itemID), slog.String("vault_id", vaultID))

	item, err := g.opGetItem(ctx, vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
	}

	return g.processItemFields(ctx, item)
}

// parseItemRef parses an item reference into vault and item components.
// Supported formats:
//   - op://vault-name-or-id/item-name-or-id
func (g *Generator) parseItemRef(itemRef itemref.Ref) (string, string, error) {
	return ParseItemRef(itemRef)
}

func (g *Generator) opGetItem(ctx context.Context, vaultID, itemID string) (onepassword.Item, error) {
	return cache.OnePasswordGetItem(ctx, g.cache, g.logger, g.client, vaultID, itemID)
}

func (g *Generator) opVaultList(ctx context.Context) ([]onepassword.VaultOverview, error) {
	return cache.OnePasswordVaultList(ctx, g.cache, g.logger, g.client)
}

func (g *Generator) opItemList(ctx context.Context, vaultID string) ([]onepassword.ItemOverview, error) {
	return cache.OnePasswordItemList(ctx, g.cache, g.logger, g.client, vaultID)
}

// resolveVaultID resolves a vault name or ID to a vault UUID.
func (g *Generator) resolveVaultID(ctx context.Context, vaultRef string) (string, error) {
	vaults, err := g.opVaultList(ctx)
	if err != nil {
		return "", fmt.Errorf("listing vaults: %w", err)
	}

	for _, v := range vaults {
		if v.ID == vaultRef || strings.EqualFold(v.Title, vaultRef) {
			return v.ID, nil
		}
	}

	return "", fmt.Errorf("%w: %q", ErrVaultNotFound, vaultRef)
}

// resolveItemID resolves an item name or ID to an item UUID within a vault.
func (g *Generator) resolveItemID(ctx context.Context, vaultID, itemRef string) (string, error) {
	items, err := g.opItemList(ctx, vaultID)
	if err != nil {
		return "", fmt.Errorf("listing items in vault %q: %w", vaultID, err)
	}

	for _, it := range items {
		if it.ID == itemRef || strings.EqualFold(it.Title, itemRef) {
			return it.ID, nil
		}
	}

	return "", fmt.Errorf("%w: %q in vault %q", ErrItemNotFound, itemRef, vaultID)
}

// processItemFields extracts environment variables from the specified section of a 1Password item.
func (g *Generator) processItemFields(
	ctx context.Context,
	item onepassword.Item,
) (<-chan EnvVar, error) {
	// Build a map of section ID -> section title.
	sectionIDToTitle := make(map[string]string, len(item.Sections))
	for _, s := range item.Sections {
		sectionIDToTitle[s.ID] = s.Title
	}

	mod := modifier.NewRegistry(
		modifier.WithRegistryLogger(g.logger),
		modifier.WithModifiers(
			modifier.NewBase64(
				modifier.WithLogger(g.logger.With("component", "base64")),
			),
			modifier.NewOnePassword(
				modifier.WithLogger(g.logger.With("component", "onepassword")),
				modifier.WithOnePasswordClient(g.client),
				modifier.WithCache(g.cache),
			),
			modifier.NewOPTmpl(
				modifier.WithLogger(g.logger.With("component", "optmpl")),
				modifier.WithOnePasswordClient(g.client),
				modifier.WithCache(g.cache),
			),
		),
	)

	envItems := make(chan EnvVar)

	go func() {
		defer close(envItems)

		for _, field := range item.Fields {
			// Only process fields in the target section.
			if field.SectionID == nil || !strings.EqualFold(sectionIDToTitle[*field.SectionID], g.section) {
				continue
			}

			// Parse the field title to extract the variable name and modifiers.
			// Format: "VAR_NAME:modifier1:modifier2:..."
			parts := strings.Split(field.Title, ":")
			varName := parts[0]
			modifiers := parts[1:]

			if !validEnvVarName.MatchString(varName) {
				g.logger.WarnContext(ctx, "skipping field with invalid environment variable name",
					slog.String("field", varName),
				)

				continue
			}

			g.logger.InfoContext(ctx, "processing field",
				slog.String("field", varName),
				slog.String("modifiers", strings.Join(modifiers, ",")),
			)

			value, err := mod.Apply(ctx, field.Value, modifiers)
			if err != nil {
				g.logger.WarnContext(ctx, "skipping field due to error",
					slog.String("field", varName),
					slogtool.ErrorAttr(err),
				)

				continue
			}

			envItems <- EnvVar{Name: varName, Value: value}
		}
	}()

	return envItems, nil
}

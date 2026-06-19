package onepassword

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/1password/onepassword-sdk-go"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
	"github.com/na4ma4/1password-direnv-tool/model"
)

const (
	splitRefParts = 2
)

var (
	ErrInvalidItemRef = errors.New("invalid item reference")
	ErrVaultNotFound  = errors.New("vault not found")
	ErrItemNotFound   = errors.New("item not found")
	validEnvVarName   = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

type Provider struct {
	client cache.OPClientFunc
	cc     cache.Cache
	logger *slog.Logger
}

func NewProvider(client cache.OPClientFunc, cc cache.Cache, logger *slog.Logger) *Provider {
	return &Provider{
		client: client,
		cc:     cc,
		logger: logger,
	}
}

func (p *Provider) Name() string {
	return "1password"
}

func (p *Provider) LookupEnvVars(ctx context.Context, ref string, section string) ([]model.EnvItem, error) {
	vaultRef, itemName, err := p.parseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("parsing item reference %q: %w", ref, err)
	}

	p.logger.DebugContext(ctx, "resolving vault", slog.String("vault", vaultRef))

	vaultID, err := p.resolveVaultID(ctx, vaultRef)
	if err != nil {
		return nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
	}

	p.logger.DebugContext(ctx, "resolving item", slog.String("item", itemName), slog.String("vault_id", vaultID))

	itemID, err := p.resolveItemID(ctx, vaultID, itemName)
	if err != nil {
		return nil, fmt.Errorf("resolving item %q: %w", itemName, err)
	}

	p.logger.DebugContext(ctx, "getting item", slog.String("item_id", itemID), slog.String("vault_id", vaultID))

	item, err := cache.OnePasswordGetItem(ctx, p.cc, p.logger, p.client, vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
	}

	return p.processItemFields(ctx, &item, section)
}

func (p *Provider) SecretResolver() model.SecretResolver {
	return &secretResolver{p: p}
}

type secretResolver struct {
	p *Provider
	// client cache.OPClientFunc
	// cc     cache.Cache
	// logger *slog.Logger
}

func (r *secretResolver) ResolveSecret(ctx context.Context, ref string) (string, error) {
	return cache.OnePasswordSecretResolve(ctx, r.p.cc, r.p.logger, r.p.client, ref)
}

func (r *secretResolver) RetrieveK8sCredential(ctx context.Context, ref string) (*model.ExecCredential, error) {
	var vaultRef, itemName string
	{
		var err error
		vaultRef, itemName, err = r.p.parseRef(ref)
		if err != nil {
			return nil, fmt.Errorf("parsing item reference %q: %w", ref, err)
		}
	}

	r.p.logger.DebugContext(ctx, "resolving vault for k8s credential", slog.String("vault", vaultRef))

	var vaultID string
	{
		var err error
		vaultID, err = r.p.resolveVaultID(ctx, vaultRef)
		if err != nil {
			return nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
		}
	}

	r.p.logger.DebugContext(
		ctx,
		"resolving item for k8s credential",
		slog.String("item", itemName),
		slog.String("vault_id", vaultID),
	)

	var itemID string
	{
		var err error
		itemID, err = r.p.resolveItemID(ctx, vaultID, itemName)
		if err != nil {
			return nil, fmt.Errorf("resolving item %q: %w", itemName, err)
		}
	}

	r.p.logger.DebugContext(
		ctx,
		"getting item for k8s credential",
		slog.String("item_id", itemID),
		slog.String("vault_id", vaultID),
	)

	var item onepassword.Item
	{
		var err error
		item, err = cache.OnePasswordGetItem(ctx, r.p.cc, r.p.logger, r.p.client, vaultID, itemID)
		if err != nil {
			return nil, fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
		}
	}

	o := model.NewExecCredential()

	if field, err := r.p.findFieldByShortRef(ctx, &item, "server"); err == nil {
		o.Spec.Cluster.Server = field.Value
	}

	{
		file, err := r.p.getFileContentByName(ctx, &item, "client.pem")
		if err != nil {
			return nil, fmt.Errorf("finding client certificate file(client.pem): %w", err)
		}

		o.Status.ClientCertificateData = string(file)
	}

	{
		file, err := r.p.getFileContentByName(ctx, &item, "client-key.pem")
		if err != nil {
			return nil, fmt.Errorf("finding client key file(client-key.pem): %w", err)
		}

		o.Status.ClientKeyData = string(file)
	}

	{
		file, err := r.p.getFileContentByName(ctx, &item, "ca.pem")
		if err != nil {
			return nil, fmt.Errorf("finding CA certificate file(ca.pem): %w", err)
		}

		o.Spec.Cluster.CertificateAuthorityData = base64.StdEncoding.EncodeToString(file)
	}

	return o, nil
}

// func (r *secretResolver) GetItem(ctx context.Context, ref string) (*model.OnePasswordItem, error) {
// 	vaultRef, itemName, err := r.p.parseRef(ref)
// 	if err != nil {
// 		return nil, fmt.Errorf("parsing item reference %q: %w", ref, err)
// 	}

// 	r.p.logger.DebugContext(ctx, "resolving vault for secret resolver", slog.String("vault", vaultRef))

// 	vaultID, err := r.p.resolveVaultID(ctx, vaultRef)
// 	if err != nil {
// 		return nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
// 	}

// 	r.p.logger.DebugContext(ctx, "resolving item for secret resolver", slog.String("item", itemName), slog.String("vault_id", vaultID))

// 	itemID, err := r.p.resolveItemID(ctx, vaultID, itemName)
// 	if err != nil {
// 		return nil, fmt.Errorf("resolving item %q: %w", itemName, err)
// 	}

// 	r.p.logger.DebugContext(ctx, "getting item for secret resolver", slog.String("item_id", itemID), slog.String("vault_id", vaultID))

// 	item, err := cache.OnePasswordGetItem(ctx, r.p.cc, r.p.logger, r.p.client, vaultID, itemID)
// 	if err != nil {
// 		return nil, fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
// 	}

// 	return r.p.convertItemToModel(&item), nil
// }

func (p *Provider) parseRef(ref string) (string, string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrInvalidItemRef, err)
	}

	if u.Scheme != "op" {
		return "", "", fmt.Errorf("%w: expected op:// scheme, got %q", ErrInvalidItemRef, u.Scheme)
	}

	vaultRef := u.Host
	itemRef := strings.TrimPrefix(u.Path, "/")

	if vaultRef == "" || itemRef == "" {
		return "", "", fmt.Errorf("%w: expected op://vault/item format", ErrInvalidItemRef)
	}

	//nolint:mnd // split into at most 2 parts, vault ref and item ref
	parts := strings.SplitN(itemRef, "/", 2)
	itemRef = parts[0]

	return vaultRef, itemRef, nil
}

func (p *Provider) resolveVaultID(ctx context.Context, vaultRef string) (string, error) {
	vaults, err := cache.OnePasswordVaultList(ctx, p.cc, p.logger, p.client)
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

func (p *Provider) resolveItemID(ctx context.Context, vaultID, itemRef string) (string, error) {
	items, err := cache.OnePasswordItemList(ctx, p.cc, p.logger, p.client, vaultID)
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

func (p *Provider) processItemFields(
	ctx context.Context,
	item *onepassword.Item,
	section string,
) ([]model.EnvItem, error) {
	sectionIDToTitle := make(map[string]string, len(item.Sections))
	for _, s := range item.Sections {
		sectionIDToTitle[s.ID] = s.Title
	}

	var result []model.EnvItem

	for _, field := range item.Fields {
		if field.SectionID == nil || !strings.EqualFold(sectionIDToTitle[*field.SectionID], section) {
			continue
		}

		parts := strings.Split(field.Title, ":")
		varName := parts[0]
		modifiers := parts[1:]

		if !validEnvVarName.MatchString(varName) {
			p.logger.WarnContext(ctx, "skipping field with invalid environment variable name",
				slog.String("field", varName),
			)

			continue
		}

		p.logger.InfoContext(ctx, "processing field",
			slog.String("field", varName),
			slog.String("modifiers", strings.Join(modifiers, ",")),
		)

		result = append(result, model.EnvItem{
			Name:      varName,
			Value:     field.Value,
			Modifiers: modifiers,
		})
	}

	return result, nil
}

func (p *Provider) findFieldByShortRef(
	_ context.Context,
	item *onepassword.Item,
	ref string,
) (onepassword.ItemField, error) {
	spRef := strings.SplitN(ref, "/", splitRefParts)
	if len(spRef) != splitRefParts {
		for _, f := range item.Fields {
			if strings.EqualFold(f.Title, ref) {
				return f, nil
			}
		}

		return onepassword.ItemField{}, fmt.Errorf("%w: no field with title %q", ErrItemNotFound, ref)
	}

	sectionID := ""
	for _, s := range item.Sections {
		if strings.EqualFold(s.Title, spRef[0]) {
			sectionID = s.ID
			break
		}
	}

	for _, f := range item.Fields {
		if f.SectionID != nil && *f.SectionID == sectionID && strings.EqualFold(f.Title, spRef[1]) {
			return f, nil
		}
	}

	return onepassword.ItemField{}, fmt.Errorf(
		"%w: no field with title %q in section %q",
		ErrItemNotFound,
		spRef[1],
		spRef[0],
	)
}

func (p *Provider) getFileContentByName(ctx context.Context, item *onepassword.Item, name string) ([]byte, error) {
	if item.Document != nil && strings.EqualFold(item.Document.Name, name) {
		d, err := cache.OnePasswordGetFileContent(ctx, p.cc, p.logger, p.client, item.VaultID, item.ID, *item.Document)
		if err != nil {
			return nil, fmt.Errorf("getting file content for document %q: %w", name, err)
		}

		return d, nil
	}

	for _, f := range item.Files {
		// log.Printf("checking file %q for name %q", f.Attributes.Name, name)
		// log.Printf("file: %+v", f)
		if strings.EqualFold(f.Attributes.Name, name) {
			d, err := cache.OnePasswordGetFileContent(
				ctx,
				p.cc,
				p.logger,
				p.client,
				item.VaultID,
				item.ID,
				f.Attributes,
			)
			if err != nil {
				return nil, fmt.Errorf("getting file content for file %q: %w", name, err)
			}

			return d, nil
		}
	}

	return nil, fmt.Errorf("%w: no file with name %q", ErrItemNotFound, name)
}

// func (p *Provider) convertItemToK8sCredential(ctx context.Context, item *onepassword.Item) (*model.ExecCredential, error) {
// 	o := &model.ExecCredential{
// 		APIVersion: "client.authentication.k8s.io/v1beta1",
// 		Kind:       "ExecCredential",
// 		Status:     model.ExecCredentialStatus{},
// 		Spec:       model.ExecCredentialSpec{},
// 	}

// 	o.Spec.Cluster.CertificateAuthorityData = base64.StdEncoding.EncodeToString()

// 	return o, nil
// }

// func (p *Provider) convertItemToModel(item *onepassword.Item) *model.OnePasswordItem {
// 	o := &model.OnePasswordItem{
// 		ID:    item.ID,
// 		Vault: &model.OnePasswordVault{ID: item.VaultID},
// 		Title: item.Title,
// 	}

// 	for _, s := range item.Sections {
// 		o.Sections = append(o.Sections, &model.OnePasswordSection{
// 			ID:    s.ID,
// 			Title: s.Title,
// 		})
// 	}

// 	for _, f := range item.Fields {
// 		var section *model.OnePasswordSection
// 		if f.SectionID != nil {
// 			section = &model.OnePasswordSection{
// 				ID: *f.SectionID,
// 			}
// 		}

// 		o.Fields = append(o.Fields, &model.OnePasswordField{
// 			ID:      f.ID,
// 			Section: section,
// 			Title:   f.Title,
// 			Value:   f.Value,
// 			Type:    string(f.FieldType),
// 		})
// 	}

// 	for _, f := range item.Files {
// 		o.Files = append(o.Files, &model.OnePasswordFile{
// 			ID:        f.Attributes.ID,
// 			Name:      f.Attributes.Name,
// 			Size:      f.Attributes.Size,
// 			SectionID: f.SectionID,
// 		})
// 	}

// 	return o
// }

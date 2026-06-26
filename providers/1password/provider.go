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

func (p *Provider) LookupEnvVars(ctx context.Context, ref string, section string) ([]model.EnvVar, error) {
	var vaultRef, itemName string
	{
		var err error
		vaultRef, itemName, _, err = p.parseRef(ref)
		if err != nil {
			return nil, fmt.Errorf("parsing item reference %q: %w", ref, err)
		}
	}

	p.logger.DebugContext(ctx, "resolving vault", slog.String("vault", vaultRef))

	fileList := &model.FileList{}

	var vaultID string
	{
		var err error
		var files *model.FileList
		vaultID, files, err = p.resolveVaultID(ctx, vaultRef)
		if err != nil {
			return nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
		}
		fileList.Merge(files)
	}

	p.logger.DebugContext(ctx, "resolving item", slog.String("item", itemName), slog.String("vault_id", vaultID))

	var itemID string
	{
		var err error
		var files *model.FileList
		itemID, files, err = p.resolveItemID(ctx, vaultID, itemName)
		if err != nil {
			return nil, fmt.Errorf("resolving item %q: %w", itemName, err)
		}
		fileList.Merge(files)
	}

	p.logger.DebugContext(ctx, "getting item", slog.String("item_id", itemID), slog.String("vault_id", vaultID))

	item, files, err := cache.OnePasswordGetItem(ctx, p.cc, p.logger, p.client, vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
	}

	return p.processItemFields(ctx, &item, fileList.Merge(files), section)
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

func (r *secretResolver) ResolveSecret(ctx context.Context, ref *model.SecretRef) error {
	r.p.logger.DebugContext(ctx, "resolving secret", slog.String("ref", ref.Value))

	var vaultRef, itemName, fieldRef string
	{
		var err error
		vaultRef, itemName, fieldRef, err = r.p.parseRef(ref.GetValue())
		if err != nil {
			return fmt.Errorf("parsing secret reference %q: %w", ref.Value, err)
		}
	}

	var vaultID string
	{
		var err error
		var files *model.FileList
		vaultID, files, err = r.p.resolveVaultID(ctx, vaultRef)
		if err != nil {
			return fmt.Errorf("resolving vault %q: %w", vaultRef, err)
		}
		ref.MergeFiles(files)
	}

	var itemID string
	{
		var err error
		var files *model.FileList
		itemID, files, err = r.p.resolveItemID(ctx, vaultID, itemName)
		if err != nil {
			return fmt.Errorf("resolving item %q: %w", itemName, err)
		}
		ref.MergeFiles(files)
	}

	var item onepassword.Item
	{
		var err error
		var files *model.FileList
		item, files, err = cache.OnePasswordGetItem(ctx, r.p.cc, r.p.logger, r.p.client, vaultID, itemID)
		if err != nil {
			return fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
		}
		ref.MergeFiles(files)
	}

	if fieldRef != "" {
		field, err := r.p.findFieldByShortRef(ctx, &item, fieldRef)
		if err != nil {
			return fmt.Errorf("finding field %q in item %q: %w", fieldRef, itemName, err)
		}
		ref.Value = field.Value
		return nil
	}

	if len(item.Fields) > 0 {
		ref.Value = item.Fields[0].Value
		return nil
	}

	return fmt.Errorf("no fields found in item %q", itemName)
}

func (r *secretResolver) RetrieveK8sCredential(
	ctx context.Context,
	ref string,
) (*model.ExecCredential, *model.FileList, error) {
	var vaultRef, itemName string
	{
		var err error
		vaultRef, itemName, _, err = r.p.parseRef(ref)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing item reference %q: %w", ref, err)
		}
	}

	r.p.logger.DebugContext(ctx, "resolving vault for k8s credential", slog.String("vault", vaultRef))
	fileList := &model.FileList{}

	var vaultID string
	{
		var err error
		var files *model.FileList
		vaultID, files, err = r.p.resolveVaultID(ctx, vaultRef)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
		}
		fileList.Merge(files)
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
		var files *model.FileList
		itemID, files, err = r.p.resolveItemID(ctx, vaultID, itemName)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving item %q: %w", itemName, err)
		}
		fileList.Merge(files)
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
		var files *model.FileList
		item, files, err = cache.OnePasswordGetItem(ctx, r.p.cc, r.p.logger, r.p.client, vaultID, itemID)
		if err != nil {
			return nil, nil, fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
		}
		fileList.Merge(files)
	}

	o := model.NewExecCredential()

	if field, err := r.p.findFieldByShortRef(ctx, &item, "server"); err == nil {
		o.Spec.Cluster.Server = field.Value
	}

	{
		file, files, err := r.p.getFileContentByName(ctx, &item, "client.pem")
		if err != nil {
			return nil, nil, fmt.Errorf("finding client certificate file(client.pem): %w", err)
		}
		fileList.Merge(files)

		o.Status.ClientCertificateData = string(file)
	}

	{
		file, files, err := r.p.getFileContentByName(ctx, &item, "client-key.pem")
		if err != nil {
			return nil, nil, fmt.Errorf("finding client key file(client-key.pem): %w", err)
		}
		fileList.Merge(files)

		o.Status.ClientKeyData = string(file)
	}

	{
		file, files, err := r.p.getFileContentByName(ctx, &item, "ca.pem")
		if err != nil {
			return nil, nil, fmt.Errorf("finding CA certificate file(ca.pem): %w", err)
		}
		fileList.Merge(files)

		o.Spec.Cluster.CertificateAuthorityData = base64.StdEncoding.EncodeToString(file)
	}

	return o, fileList, nil
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

func (p *Provider) parseRef(ref string) (string, string, string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return "", "", "", fmt.Errorf("%w: %w", ErrInvalidItemRef, err)
	}

	if u.Scheme != "op" {
		return "", "", "", fmt.Errorf("%w: expected op:// scheme, got %q", ErrInvalidItemRef, u.Scheme)
	}

	vaultRef := u.Host
	itemPath := strings.TrimPrefix(u.Path, "/")

	if vaultRef == "" || itemPath == "" {
		return "", "", "", fmt.Errorf("%w: expected op://vault/item format", ErrInvalidItemRef)
	}

	//nolint:mnd // split into at most 2 parts, vault ref and item ref / field ref
	parts := strings.SplitN(itemPath, "/", 2)
	itemRef := parts[0]
	var fieldRef string
	if len(parts) > 1 {
		fieldRef = parts[1]
	}

	return vaultRef, itemRef, fieldRef, nil
}

func (p *Provider) resolveVaultID(ctx context.Context, vaultRef string) (string, *model.FileList, error) {
	vaults, files, err := cache.OnePasswordVaultList(ctx, p.cc, p.logger, p.client)
	if err != nil {
		return "", nil, fmt.Errorf("listing vaults: %w", err)
	}

	// Special case for "Personal" vault, which is commonly used and can be identified by the "personal" tag or by its title. This allows users to reference the Personal vault without needing to know its ID.
	// The check is case-insensitive to improve usability.
	// If the vaultRef is "Personal" (case-insensitive), we look for a vault with the "personal" tag or with the title "Personal".
	switch strings.ToLower(vaultRef) {
	case "personal", "private":
		for _, v := range vaults {
			if strings.EqualFold(v.Title, "Personal") || strings.EqualFold(v.Title, "Private") ||
				v.VaultType == onepassword.VaultTypePersonal {
				return v.ID, files, nil
			}
		}
	}

	for _, v := range vaults {
		if v.ID == vaultRef || strings.EqualFold(v.Title, vaultRef) {
			return v.ID, files, nil
		}
	}

	return "", nil, fmt.Errorf("%w: %q", ErrVaultNotFound, vaultRef)
}

func (p *Provider) resolveItemID(ctx context.Context, vaultID, itemRef string) (string, *model.FileList, error) {
	items, files, err := cache.OnePasswordItemList(ctx, p.cc, p.logger, p.client, vaultID)
	if err != nil {
		return "", nil, fmt.Errorf("listing items in vault %q: %w", vaultID, err)
	}

	for _, it := range items {
		if it.ID == itemRef || strings.EqualFold(it.Title, itemRef) {
			return it.ID, files, nil
		}
	}

	return "", nil, fmt.Errorf("%w: %q in vault %q", ErrItemNotFound, itemRef, vaultID)
}

func (p *Provider) processItemFields(
	ctx context.Context,
	item *onepassword.Item,
	files *model.FileList,
	section string,
) ([]model.EnvVar, error) {
	sectionIDToTitle := make(map[string]string, len(item.Sections))
	for _, s := range item.Sections {
		sectionIDToTitle[s.ID] = s.Title
	}

	var result []model.EnvVar

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

		result = append(result, &model.EnvItem{
			Name:      varName,
			Value:     field.Value,
			FileList:  files,
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

func (p *Provider) getFileContentByName(
	ctx context.Context,
	item *onepassword.Item,
	name string,
) ([]byte, *model.FileList, error) {
	if item.Document != nil && strings.EqualFold(item.Document.Name, name) {
		d, files, err := cache.OnePasswordGetFileContent(
			ctx,
			p.cc,
			p.logger,
			p.client,
			item.VaultID,
			item.ID,
			*item.Document,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("getting file content for document %q: %w", name, err)
		}

		return d, files, nil
	}

	for _, f := range item.Files {
		// log.Printf("checking file %q for name %q", f.Attributes.Name, name)
		// log.Printf("file: %+v", f)
		if strings.EqualFold(f.Attributes.Name, name) {
			d, files, err := cache.OnePasswordGetFileContent(
				ctx,
				p.cc,
				p.logger,
				p.client,
				item.VaultID,
				item.ID,
				f.Attributes,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("getting file content for file %q: %w", name, err)
			}

			return d, files, nil
		}
	}

	return nil, nil, fmt.Errorf("%w: no file with name %q", ErrItemNotFound, name)
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

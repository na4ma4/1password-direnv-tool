package protonpass

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
	"github.com/na4ma4/1password-direnv-tool/model"
)

const itemRefPartsCount = 2

var (
	ErrInvalidItemRef = errors.New("invalid item reference")
	ErrVaultNotFound  = errors.New("vault not found")
)

var validEnvVarName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Provider struct {
	cc          cache.Cache
	logger      *slog.Logger
	passCLIPath string
}

func NewProvider(logger *slog.Logger, cc cache.Cache) *Provider {
	return &Provider{logger: logger, cc: cc, passCLIPath: findPassCLI()}
}

func (p *Provider) Name() string {
	return "protonpass"
}

func (p *Provider) LookupEnvVars(ctx context.Context, ref string, section string) ([]model.EnvVar, error) {
	var vaultRef, itemRef string
	{
		var err error
		vaultRef, itemRef, err = p.parseRef(ref)
		if err != nil {
			return nil, fmt.Errorf("parsing item reference %q: %w", ref, err)
		}
	}

	p.logger.DebugContext(ctx, "listing vaults")
	fileList := &model.FileList{}

	var shareID string
	{
		var err error
		var files *model.FileList
		shareID, files, err = p.resolveVaultShareID(ctx, vaultRef)
		if err != nil {
			return nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
		}
		fileList.Merge(files)
	}

	p.logger.DebugContext(ctx, "viewing item",
		slog.String("vault", vaultRef),
		slog.String("item", itemRef),
	)

	var viewResp passItemViewResponse
	{
		var err error
		var files *model.FileList
		viewResp, files, err = p.viewItem(ctx, shareID, itemRef)
		if err != nil {
			return nil, fmt.Errorf("viewing item %q: %w", itemRef, err)
		}
		fileList.Merge(files)
	}

	return p.processItemFields(ctx, fileList, viewResp.Item, section)
}

func (p *Provider) SecretResolver() model.SecretResolver {
	return &secretResolver{p: p}
}

type secretResolver struct {
	p *Provider
}

func (r *secretResolver) ResolveSecret(ctx context.Context, ref *model.SecretRef) error {
	return r.p.resolveSecret(ctx, ref)
}

func (r *secretResolver) retrieveK8sItem(
	ctx context.Context,
	ref string,
) (string, passItemViewResponse, *model.FileList, error) {
	vaultRef, itemRef, err := r.p.parseRef(ref)
	if err != nil {
		return "", passItemViewResponse{}, nil, fmt.Errorf("parsing item reference %q: %w", ref, err)
	}

	r.p.logger.DebugContext(ctx, "resolving vault for k8s credential", slog.String("vault", vaultRef))

	fileList := &model.FileList{}
	shareID, files, err := r.p.resolveVaultShareID(ctx, vaultRef)
	if err != nil {
		return "", passItemViewResponse{}, nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
	}
	fileList.Merge(files)

	r.p.logger.DebugContext(ctx, "viewing item for k8s credential",
		slog.String("vault", vaultRef), slog.String("item", itemRef),
	)

	viewResp, files, err := r.p.viewItem(ctx, shareID, itemRef)
	if err != nil {
		return "", passItemViewResponse{}, nil, fmt.Errorf("viewing item %q: %w", itemRef, err)
	}
	fileList.Merge(files)

	return shareID, viewResp, fileList, nil
}

func (r *secretResolver) retrieveK8sAttachment(
	ctx context.Context,
	shareID, itemID, attachmentID string,
	fileList *model.FileList,
) ([]byte, *model.FileList, error) {
	if attachmentID == "" {
		return nil, fileList, nil
	}

	data, files, err := r.p.downloadAttachment(ctx, shareID, itemID, attachmentID)
	if err != nil {
		return nil, nil, err
	}

	return data, fileList.Merge(files), nil
}

func (r *secretResolver) RetrieveK8sCredential(
	ctx context.Context,
	ref string,
) (*model.ExecCredential, *model.FileList, error) {
	var err error
	shareID, viewResp, fileList, err := r.retrieveK8sItem(ctx, ref)
	if err != nil {
		return nil, nil, err
	}

	o := model.NewExecCredential()

	var itemContent passItemNote
	if err = json.Unmarshal(viewResp.Item.Content, &itemContent); err != nil {
		return nil, nil, fmt.Errorf("parsing item content: %w", err)
	}

	for _, ef := range itemContent.ExtraFields {
		if strings.EqualFold(ef.Name, "server") {
			o.Spec.Cluster.Server = extractExtraFieldValue(ef.Content)
			break
		}
	}

	var clientPemID, clientKeyPemID, caPemID string
	for _, att := range viewResp.Attachments {
		switch strings.ToLower(att.Content.Name) {
		case "client.pem":
			clientPemID = att.ID
		case "client-key.pem":
			clientKeyPemID = att.ID
		case "ca.pem":
			caPemID = att.ID
		}
	}

	var data []byte
	var fl *model.FileList

	if data, fl, err = r.retrieveK8sAttachment(ctx, shareID, viewResp.Item.ID, clientPemID, fileList); err != nil {
		return nil, nil, fmt.Errorf("downloading client.pem: %w", err)
	}
	fileList = fl
	o.Status.ClientCertificateData = string(data)

	if data, fl, err = r.retrieveK8sAttachment(ctx, shareID, viewResp.Item.ID, clientKeyPemID, fileList); err != nil {
		return nil, nil, fmt.Errorf("downloading client-key.pem: %w", err)
	}
	fileList = fl
	o.Status.ClientKeyData = string(data)

	if data, fl, err = r.retrieveK8sAttachment(ctx, shareID, viewResp.Item.ID, caPemID, fileList); err != nil {
		return nil, nil, fmt.Errorf("downloading ca.pem: %w", err)
	}
	fileList = fl
	o.Spec.Cluster.CertificateAuthorityData = base64.StdEncoding.EncodeToString(data)

	return o, fileList, nil
}

func (p *Provider) resolveSecret(ctx context.Context, ref *model.SecretRef) error {
	value, files, err := cache.ProtonPassResolveSecret(ctx, p.cc, p.logger, p.runPassCLIOutput, ref.GetValue())
	if err != nil {
		return fmt.Errorf("resolving secret: %w", err)
	}
	ref.MergeFiles(files)
	ref.Value = value
	return nil
}

func (p *Provider) parseRef(ref string) (string, string, error) {
	var u *url.URL
	{
		var err error
		u, err = url.Parse(ref)
		if err != nil {
			return "", "", fmt.Errorf("%w: %w", ErrInvalidItemRef, err)
		}
	}

	if u.Scheme != "pass" {
		return "", "", fmt.Errorf("%w: expected pass:// scheme, got %q", ErrInvalidItemRef, u.Scheme)
	}

	vaultRef := u.Host
	itemRef := strings.TrimPrefix(u.Path, "/")

	if vaultRef == "" || itemRef == "" {
		return "", "", fmt.Errorf("%w: expected pass://vault/item format", ErrInvalidItemRef)
	}

	parts := strings.SplitN(itemRef, "/", itemRefPartsCount)
	itemRef = parts[0]

	return vaultRef, itemRef, nil
}

func (p *Provider) resolveVaultShareID(ctx context.Context, vaultRef string) (string, *model.FileList, error) {
	fileList := &model.FileList{}

	var vaultsJSON string
	{
		var err error
		var files *model.FileList
		vaultsJSON, files, err = cache.ProtonPassVaultList(ctx, p.cc, p.logger, p.runPassCLIOutput)
		if err != nil {
			return "", nil, fmt.Errorf("listing vaults: %w", err)
		}
		fileList.Merge(files)
	}

	var vaultList struct {
		Vaults []struct {
			Name    string `json:"name"`
			VaultID string `json:"vault_id"`
			ShareID string `json:"share_id"`
		} `json:"vaults"`
	}

	if err := json.Unmarshal([]byte(vaultsJSON), &vaultList); err != nil {
		return "", nil, fmt.Errorf("parsing vault list: %w", err)
	}

	for _, v := range vaultList.Vaults {
		if v.ShareID == vaultRef || strings.EqualFold(v.Name, vaultRef) {
			return v.ShareID, fileList, nil
		}
	}

	return "", nil, fmt.Errorf("%w: %q", ErrVaultNotFound, vaultRef)
}

func (p *Provider) viewItem(
	ctx context.Context,
	shareID string,
	itemRef string,
) (passItemViewResponse, *model.FileList, error) {
	var itemJSON string
	var fileList *model.FileList
	{
		var err error
		itemJSON, fileList, err = cache.ProtonPassViewItem(ctx, p.cc, p.logger, p.runPassCLIOutput, shareID, itemRef)
		if err != nil {
			return passItemViewResponse{}, nil, fmt.Errorf("viewing item: %w", err)
		}
	}

	var viewResp passItemViewResponse
	if err := json.Unmarshal([]byte(itemJSON), &viewResp); err != nil {
		return passItemViewResponse{}, nil, fmt.Errorf("parsing item JSON: %w", err)
	}

	return viewResp, fileList, nil
}

type passItemData struct {
	ID         string          `json:"id"`
	ShareID    string          `json:"share_id"`
	VaultID    string          `json:"vault_id"`
	Content    json.RawMessage `json:"content"`
	State      string          `json:"state"`
	Flags      []string        `json:"flags"`
	CreateTime string          `json:"create_time"`
	ModifyTime string          `json:"modify_time"`
}

type passItemViewResponse struct {
	Item        passItemData     `json:"item"`
	Attachments []passAttachment `json:"attachments"`
}

type passAttachment struct {
	ID      string                `json:"id"`
	Content passAttachmentContent `json:"content"`
}

type passAttachmentContent struct {
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
}

type passItemNote struct {
	Title       string           `json:"title"`
	Note        string           `json:"note"`
	ExtraFields []passExtraField `json:"extra_fields"`
	Content     json.RawMessage  `json:"content"`
}

type passExtraField struct {
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"`
}

type passLoginContent struct {
	Email    string   `json:"email"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	TotpURI  string   `json:"totp_uri"`
	URLs     []string `json:"urls"`
}

type passCustomSection struct {
	SectionName   string           `json:"section_name"`
	SectionFields []passExtraField `json:"section_fields"`
}

type passCustomContent struct {
	Sections []passCustomSection `json:"sections"`
}

func (p *Provider) processItemFields(
	ctx context.Context,
	fileList *model.FileList,
	item passItemData,
	section string,
) ([]model.EnvVar, error) {
	var result []model.EnvVar

	fields := p.extractAllFields(ctx, item)

	for _, f := range fields {
		if section != "" && f.section != "" && !strings.EqualFold(f.section, section) {
			continue
		}

		parts := strings.Split(f.name, ":")
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
			FileList:  fileList,
			Value:     f.value,
			Modifiers: modifiers,
		})
	}

	return result, nil
}

type extractedField struct {
	name    string
	value   string
	section string
}

func (p *Provider) extractAllFields(ctx context.Context, item passItemData) []extractedField {
	var fields []extractedField

	var itemContent passItemNote
	if err := json.Unmarshal(item.Content, &itemContent); err != nil {
		p.logger.WarnContext(ctx, "failed to parse item content", slog.String("error", err.Error()))
		return nil
	}

	// if itemContent.Title != "" {
	// 	fields = append(fields, extractedField{name: "title", value: itemContent.Title, section: "hidden"})
	// }
	if itemContent.Note != "" {
		// fields = append(fields, extractedField{name: "note", value: itemContent.Note, section: "hidden"})
		scanner := bufio.NewScanner(strings.NewReader(itemContent.Note))

		// Loop through each line until the end of the string
		for scanner.Scan() {
			line := scanner.Text() // Extracts the current line as a string
			spline := strings.SplitN(line, "=", itemRefPartsCount)
			if len(spline) == itemRefPartsCount {
				fields = append(fields, extractedField{name: spline[0], value: spline[1], section: "Environment"})
			}
		}

		if err := scanner.Err(); err != nil {
			return nil
		}
	}

	for _, ef := range itemContent.ExtraFields {
		fields = append(fields, extractedField{
			name:  ef.Name,
			value: extractExtraFieldValue(ef.Content),
		})
	}

	fields = append(fields, p.extractTypeSpecificFields(item.Content)...)

	return fields
}

func (p *Provider) extractTypeSpecificFields(raw json.RawMessage) []extractedField {
	if len(raw) == 0 {
		return nil
	}

	var typeMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &typeMap); err != nil {
		return nil
	}

	for typeName, data := range typeMap {
		switch typeName {
		case "login":
			return extractLoginFields(data)
		case "custom":
			return extractCustomFields(data)
		}
	}

	return nil
}

func extractLoginFields(raw json.RawMessage) []extractedField {
	var login passLoginContent
	if err := json.Unmarshal(raw, &login); err != nil {
		return nil
	}

	var fields []extractedField
	if login.Email != "" {
		fields = append(fields, extractedField{name: "email", value: login.Email})
	}
	if login.Username != "" {
		fields = append(fields, extractedField{name: "username", value: login.Username})
	}
	if login.Password != "" {
		fields = append(fields, extractedField{name: "password", value: login.Password})
	}
	if login.TotpURI != "" {
		fields = append(fields, extractedField{name: "totp_uri", value: login.TotpURI})
	}

	return fields
}

func extractCustomFields(raw json.RawMessage) []extractedField {
	var custom passCustomContent
	if err := json.Unmarshal(raw, &custom); err != nil {
		return nil
	}

	var fields []extractedField
	for _, s := range custom.Sections {
		for _, f := range s.SectionFields {
			fields = append(fields, extractedField{
				name:    f.Name,
				value:   extractExtraFieldValue(f.Content),
				section: s.SectionName,
			})
		}
	}

	return fields
}

func extractExtraFieldValue(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var valMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &valMap); err != nil {
		return strings.Trim(string(raw), `"`)
	}

	for _, kind := range []string{"Text", "Hidden", "Totp"} {
		if v, ok := valMap[kind]; ok {
			return strings.Trim(string(v), `"`)
		}
	}

	if v, ok := valMap["Timestamp"]; ok {
		return strings.Trim(string(v), `"`)
	}

	return ""
}

func (p *Provider) downloadAttachment(
	ctx context.Context,
	shareID, itemID, attachmentID string,
) ([]byte, *model.FileList, error) {
	return cache.ProtonPassDownloadAttachment(
		ctx,
		p.cc,
		p.logger,
		p.downloadAttachmentUncached,
		shareID,
		itemID,
		attachmentID,
	)
}

func (p *Provider) downloadAttachmentUncached(
	ctx context.Context,
	shareID, itemID, attachmentID string,
) ([]byte, *model.FileList, error) {
	var tmpFile *os.File
	{
		var err error
		tmpFile, err = os.CreateTemp("", "pass-attachment-*")
		if err != nil {
			return nil, nil, fmt.Errorf("creating temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())
	}
	if err := tmpFile.Close(); err != nil {
		return nil, nil, fmt.Errorf("closing temp file: %w", err)
	}

	if _, err := p.runPassCLIOutput(ctx,
		"item", "attachment", "download",
		"--share-id", shareID,
		"--item-id", itemID,
		"--attachment-id", attachmentID,
		"--output", tmpFile.Name(),
	); err != nil {
		return nil, nil, err
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, nil, fmt.Errorf("reading downloaded attachment: %w", err)
	}

	return data, nil, nil
}

var errBadArg = errors.New("invalid argument")

func safePassArg(arg string) error {
	for _, r := range arg {
		if r < 0x20 || r == 0x7f {
			return errBadArg
		}
	}
	return nil
}

func (p *Provider) runPassCLIOutput(ctx context.Context, args ...string) (string, error) {
	for _, arg := range args {
		if err := safePassArg(arg); err != nil {
			return "", fmt.Errorf("%w: %q", err, arg)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, p.passCLIPath, args...) //nolint:gosec // args validated by safePassArg above
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pass-cli error: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

package modifier

import (
	"context"
	"regexp"
	"strings"

	"github.com/na4ma4/1password-direnv-tool/internal/cache"
)

var onePasswordTemplatePattern = regexp.MustCompile(`\{\{\s*"?(op://[^\s{}]+)"?\s*\}\}`)

type OnePasswordTemplateModifier struct {
	opts *options
}

func NewOPTmpl(opts ...optionsFunc) *OnePasswordTemplateModifier {
	options := &options{}

	for _, opt := range opts {
		opt(options)
	}

	options.applyDefaults()

	if options.client == nil {
		panic("OnePassword client is required for OnePasswordTemplateModifier")
	}

	return &OnePasswordTemplateModifier{opts: options}
}

func (m *OnePasswordTemplateModifier) Tags() Tags {
	return Tags{"op-tmpl", "optmpl", "1password-template"}
}

func (m *OnePasswordTemplateModifier) opSecretResolve(ctx context.Context, secretRef string) (string, error) {
	return cache.OnePasswordSecretResolve(ctx, m.opts.cache, m.opts.logger, m.opts.client, secretRef)
}

// Apply searches through the supplied value for any occurrences of {{ op://vault/item/field }}
// and replaces them with the resolved value from 1Password.
func (m *OnePasswordTemplateModifier) Apply(ctx context.Context, value string) (string, error) {
	matches := onePasswordTemplatePattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	var out strings.Builder
	last := 0

	for _, match := range matches {
		fullStart := match[0]
		fullEnd := match[1]
		secretStart := match[2]
		secretEnd := match[3]

		out.WriteString(value[last:fullStart])

		secretRef := value[secretStart:secretEnd]
		resolved, err := m.opSecretResolve(ctx, secretRef)
		if err != nil {
			return "", err
		}

		out.WriteString(resolved)
		last = fullEnd
	}

	out.WriteString(value[last:])

	return out.String(), nil
}

package modifier

import (
	"context"
	"regexp"
	"strings"

	"github.com/na4ma4/1password-direnv-tool/model"
)

type templateModifier struct {
	pattern  *regexp.Regexp
	tags     model.Tags
	resolver model.SecretResolver
	opts     *options
}

func newTemplateModifier(
	resolver model.SecretResolver,
	pattern *regexp.Regexp,
	tags model.Tags,
	panicMsg string,
	opts ...optionsFunc,
) templateModifier {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}
	options.applyDefaults()

	if resolver == nil {
		panic(panicMsg)
	}

	return templateModifier{resolver: resolver, opts: options, pattern: pattern, tags: tags}
}

func (m *templateModifier) Apply(ctx context.Context, value string) (string, error) {
	matches := m.pattern.FindAllStringSubmatchIndex(value, -1)
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
		resolved, err := m.resolver.ResolveSecret(ctx, secretRef)
		if err != nil {
			return "", err
		}

		out.WriteString(resolved)
		last = fullEnd
	}

	out.WriteString(value[last:])

	return out.String(), nil
}

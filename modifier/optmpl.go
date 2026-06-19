package modifier

import (
	"regexp"

	"github.com/na4ma4/1password-direnv-tool/model"
)

var onePasswordTemplatePattern = regexp.MustCompile(`\{\{\s*"?(op://[^\s{}]+)"?\s*\}\}`)

type OnePasswordTemplateModifier struct {
	templateModifier
}

func NewOPTmpl(resolver model.SecretResolver, opts ...optionsFunc) *OnePasswordTemplateModifier {
	m := newTemplateModifier(resolver, onePasswordTemplatePattern,
		model.Tags{"op-tmpl", "optmpl", "1password-template"},
		"SecretResolver is required for OnePasswordTemplateModifier",
		opts...,
	)
	return &OnePasswordTemplateModifier{templateModifier: m}
}

func (m *OnePasswordTemplateModifier) Tags() model.Tags {
	return m.templateModifier.tags
}

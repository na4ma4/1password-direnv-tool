package modifier

import (
	"regexp"

	"github.com/na4ma4/1password-direnv-tool/model"
)

var protonPassTemplatePattern = regexp.MustCompile(`\{\{\s*"?(pass://[^\s{}]+)"?\s*\}\}`)

type ProtonPassTemplateModifier struct {
	templateModifier
}

func NewPassTmpl(resolver model.SecretResolver, opts ...optionsFunc) *ProtonPassTemplateModifier {
	m := newTemplateModifier(resolver, protonPassTemplatePattern,
		model.Tags{"pass-tmpl", "passtmpl", "protonpass-template"},
		"SecretResolver is required for ProtonPassTemplateModifier",
		opts...,
	)
	return &ProtonPassTemplateModifier{templateModifier: m}
}

func (m *ProtonPassTemplateModifier) Tags() model.Tags {
	return m.templateModifier.tags
}

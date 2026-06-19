package openv

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/na4ma4/go-slogtool"

	"github.com/na4ma4/1password-direnv-tool/internal/itemref"
	"github.com/na4ma4/1password-direnv-tool/model"
	"github.com/na4ma4/1password-direnv-tool/modifier"
)

type Generator struct {
	provider     model.Provider
	opResolver   model.SecretResolver
	passResolver model.SecretResolver
	section      string
	logger       *slog.Logger
}

func New(
	provider model.Provider,
	opResolver, passResolver model.SecretResolver,
	section string,
	logger *slog.Logger,
) *Generator {
	return &Generator{
		provider:     provider,
		opResolver:   opResolver,
		passResolver: passResolver,
		section:      section,
		logger:       logger,
	}
}

func (g *Generator) GetEnvVars(ctx context.Context, itemRef itemref.Ref) (<-chan model.EnvVar, error) {
	g.logger.DebugContext(ctx, "looking up environment variables",
		slog.String("ref", itemRef.String()),
		slog.String("section", g.section),
	)

	items, err := g.provider.LookupEnvVars(ctx, itemRef.String(), g.section)
	if err != nil {
		return nil, fmt.Errorf("looking up env vars: %w", err)
	}

	mods := []model.Modifier{
		modifier.NewBase64(
			modifier.WithLogger(g.logger.With("component", "base64")),
		),
		modifier.NewBase32(
			modifier.WithLogger(g.logger.With("component", "base32")),
		),
		modifier.NewOnePassword(g.opResolver,
			modifier.WithLogger(g.logger.With("component", "onepassword")),
		),
		modifier.NewProtonPass(g.passResolver,
			modifier.WithLogger(g.logger.With("component", "protonpass")),
		),
		modifier.NewOPTmpl(g.opResolver,
			modifier.WithLogger(g.logger.With("component", "optmpl")),
		),
		modifier.NewPassTmpl(g.passResolver,
			modifier.WithLogger(g.logger.With("component", "passtmpl")),
		),
	}

	mod := modifier.NewRegistry(
		modifier.WithRegistryLogger(g.logger),
		modifier.WithModifiers(mods...),
	)

	envVars := make(chan model.EnvVar)

	go func() {
		defer close(envVars)

		for _, item := range items {
			val, applyErr := mod.Apply(ctx, item.Value, item.Modifiers)
			if applyErr != nil {
				g.logger.WarnContext(ctx, "skipping field due to error",
					slog.String("field", item.Name),
					slogtool.ErrorAttr(applyErr),
				)

				continue
			}

			envVars <- model.EnvVar{Name: item.Name, Value: val}
		}
	}()

	return envVars, nil
}

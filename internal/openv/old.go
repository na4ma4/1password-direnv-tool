package openv

// // GetEnvVars retrieves environment variables from a 1Password item.
// // The itemRef should be in the format "op://vault-name-or-id/item-name-or-id".
// // The section parameter specifies the section name to read from (e.g. "Environment").
// func GetEnvVars(
// 	ctx context.Context,
// 	client *onepassword.Client,
// 	cache cache.Cache,
// 	itemRef itemref.Ref,
// 	section string,
// 	logger *slog.Logger,
// ) ([]EnvVar, error) {
// 	vaultRef, itemName, err := ParseItemRef(itemRef)
// 	if err != nil {
// 		return nil, fmt.Errorf("parsing item reference %q: %w", itemRef, err)
// 	}

// 	logger.DebugContext(ctx, "Resolving vault", slog.String("vault", vaultRef))

// 	vaultID, err := resolveVaultID(ctx, client, vaultRef)
// 	if err != nil {
// 		return nil, fmt.Errorf("resolving vault %q: %w", vaultRef, err)
// 	}

// 	logger.DebugContext(ctx, "Resolving item", slog.String("item", itemName), slog.String("vault_id", vaultID))

// 	itemID, err := resolveItemID(ctx, client, vaultID, itemName)
// 	if err != nil {
// 		return nil, fmt.Errorf("resolving item %q: %w", itemName, err)
// 	}

// 	logger.DebugContext(ctx, "Getting item", slog.String("item_id", itemID), slog.String("vault_id", vaultID))

// 	item, err := client.Items().Get(ctx, vaultID, itemID)
// 	if err != nil {
// 		return nil, fmt.Errorf("getting item %q from vault %q: %w", itemID, vaultID, err)
// 	}

// 	return processItemFields(ctx, client, item, section, logger)
// }

// // resolveVaultID resolves a vault name or ID to a vault UUID.
// func resolveVaultID(ctx context.Context, client *onepassword.Client, vaultRef string) (string, error) {

// 	vaults, err := client.Vaults().List(ctx)
// 	if err != nil {
// 		return "", fmt.Errorf("listing vaults: %w", err)
// 	}

// 	for _, v := range vaults {
// 		if v.ID == vaultRef || strings.EqualFold(v.Title, vaultRef) {
// 			return v.ID, nil
// 		}
// 	}

// 	return "", fmt.Errorf("%w: %q", ErrVaultNotFound, vaultRef)
// }

// // resolveItemID resolves an item name or ID to an item UUID within a vault.
// func resolveItemID(ctx context.Context, client *onepassword.Client, vaultID, itemRef string) (string, error) {
// 	items, err := client.Items().List(ctx, vaultID)
// 	if err != nil {
// 		return "", fmt.Errorf("listing items in vault %q: %w", vaultID, err)
// 	}

// 	for _, it := range items {
// 		if it.ID == itemRef || strings.EqualFold(it.Title, itemRef) {
// 			return it.ID, nil
// 		}
// 	}

// 	return "", fmt.Errorf("%w: %q in vault %q", ErrItemNotFound, itemRef, vaultID)
// }

// // processItemFields extracts environment variables from the specified section of a 1Password item.
// func processItemFields(
// 	ctx context.Context,
// 	client *onepassword.Client,
// 	item onepassword.Item,
// 	sectionName string,
// 	logger *slog.Logger,
// ) ([]EnvVar, error) {
// 	// Build a map of section ID -> section title.
// 	sectionIDToTitle := make(map[string]string, len(item.Sections))
// 	for _, s := range item.Sections {
// 		sectionIDToTitle[s.ID] = s.Title
// 	}

// 	mod := modifier.NewRegistry(
// 		modifier.WithRegistryLogger(logger),
// 		modifier.WithModifiers(
// 			modifier.NewBase64(
// 				modifier.WithLogger(logger.With("component", "base64")),
// 			),
// 			modifier.NewOnePassword(
// 				modifier.WithLogger(logger.With("component", "onepassword")),
// 				modifier.WithOnePasswordClient(client),
// 			),
// 			modifier.NewOPTmpl(
// 				modifier.WithLogger(logger.With("component", "optmpl")),
// 				modifier.WithOnePasswordClient(client),
// 			),
// 		),
// 	)

// 	var envVars []EnvVar

// 	for _, field := range item.Fields {
// 		// Only process fields in the target section.
// 		if field.SectionID == nil {
// 			continue
// 		}

// 		if !strings.EqualFold(sectionIDToTitle[*field.SectionID], sectionName) {
// 			continue
// 		}

// 		// Parse the field title to extract the variable name and modifiers.
// 		// Format: "VAR_NAME:modifier1:modifier2:..."
// 		parts := strings.Split(field.Title, ":")
// 		varName := parts[0]
// 		modifiers := parts[1:]

// 		value, err := mod.Apply(ctx, field.Value, modifiers)
// 		if err != nil {
// 			logger.WarnContext(ctx, "Skipping field due to error",
// 				slog.String("field", varName),
// 				slogtool.ErrorAttr(err),
// 			)

// 			continue
// 		}

// 		envVars = append(envVars, EnvVar{Name: varName, Value: value})
// 	}

// 	return envVars, nil
// }

// // applyModifiers applies a sequence of modifiers to a field value.
// func applyModifiers(
// 	ctx context.Context,
// 	client *onepassword.Client,
// 	fieldName string,
// 	value string,
// 	modifiers []string,
// 	logger *slog.Logger,
// ) (string, error) {
// 	for _, modifier := range modifiers {
// 		var err error

// 		switch modifier {
// 		case "1password":
// 			logger.DebugContext(ctx, "Applying 1password modifier",
// 				slog.String("field", fieldName),
// 				slog.String("value", value),
// 			)

// 			value, err = client.Secrets().Resolve(ctx, value)
// 			if err != nil {
// 				return "", fmt.Errorf("resolving 1password secret reference %q for field %q: %w", value, fieldName, err)
// 			}

// 		case "b64", "base64":
// 			logger.DebugContext(ctx, "Applying base64 modifier", slog.String("field", fieldName))
// 			value = base64.StdEncoding.EncodeToString([]byte(value))

// 		default:
// 			logger.WarnContext(ctx, "Unknown modifier, skipping",
// 				slog.String("field", fieldName),
// 				slog.String("modifier", modifier),
// 			)
// 		}
// 	}

// 	return value, nil
// }

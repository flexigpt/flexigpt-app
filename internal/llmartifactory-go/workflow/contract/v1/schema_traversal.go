package workflowv1

// walkSchema visits Draft 2020-12 subschema positions. It does not interpret
// instance data inside const, enum, default, or examples as schema keywords.
func walkSchema(
	value any,
	visit func(map[string]any) error,
) error {
	object, ok := value.(map[string]any)
	if !ok {
		// Boolean schemas have no child schemas.
		return nil
	}
	if err := visit(object); err != nil {
		return err
	}

	for _, keyword := range []string{
		"$defs",
		"definitions",
		"properties",
		"patternProperties",
		"dependentSchemas",
	} {
		children, _ := object[keyword].(map[string]any)
		for _, name := range sortedKeys(children) {
			if err := walkSchema(children[name], visit); err != nil {
				return err
			}
		}
	}

	for _, keyword := range []string{
		"items",
		"contains",
		"additionalProperties",
		"unevaluatedProperties",
		"unevaluatedItems",
		"propertyNames",
		"not",
		"if",
		"then",
		"else",
		"contentSchema",
	} {
		if err := walkSchema(object[keyword], visit); err != nil {
			return err
		}
	}

	for _, keyword := range []string{
		"allOf",
		"anyOf",
		"oneOf",
		"prefixItems",
	} {
		children, _ := object[keyword].([]any)
		for _, child := range children {
			if err := walkSchema(child, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

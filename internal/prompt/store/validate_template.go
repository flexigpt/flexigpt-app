package store

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
)

// Regular expressions.
var (
	placeholderRE = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}`)
)

// validateTemplate performs a structural and referential integrity check.
func validateTemplate(tpl *spec.PromptTemplate) error {
	if tpl == nil {
		return errors.New("template is nil")
	}
	if tpl.SchemaVersion != spec.SchemaVersion {
		return fmt.Errorf(
			"schemaVersion %q does not match expected %q",
			tpl.SchemaVersion,
			spec.SchemaVersion,
		)
	}

	if err := bundleitemutils.ValidateItemSlug(tpl.Slug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}
	if err := bundleitemutils.ValidateItemVersion(tpl.Version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	switch tpl.Kind {
	case spec.PromptTemplateKindInstructionsOnly, spec.PromptTemplateKindGeneric:
		// Ok.
	default:
		return fmt.Errorf("invalid kind %q", tpl.Kind)
	}

	if strings.TrimSpace(tpl.DisplayName) == "" {
		return errors.New("displayName is empty")
	}
	if tpl.CreatedAt.IsZero() {
		return errors.New("createdAt is zero")
	}
	if tpl.ModifiedAt.IsZero() {
		return errors.New("modifiedAt is zero")
	}
	if len(tpl.Blocks) == 0 {
		return errors.New("at least one message block required")
	}

	allowedRoles := allowedRolesForKind(tpl.Kind)
	blockIDs := map[spec.MessageBlockID]struct{}{}
	placeholders := map[string]struct{}{}

	for i, b := range tpl.Blocks {
		if _, ok := allowedRoles[b.Role]; !ok {
			return fmt.Errorf("blocks[%d]: role %q not allowed for kind %q", i, b.Role, tpl.Kind)
		}
		if strings.TrimSpace(b.Content) == "" {
			return fmt.Errorf("blocks[%d]: content is empty", i)
		}
		if strings.TrimSpace(string(b.ID)) == "" {
			return fmt.Errorf("blocks[%d]: id is empty", i)
		}
		if _, dup := blockIDs[b.ID]; dup {
			return fmt.Errorf("duplicate block id %q", b.ID)
		}
		blockIDs[b.ID] = struct{}{}

		for _, m := range placeholderRE.FindAllStringSubmatch(b.Content, -1) {
			if len(m) == 2 {
				placeholders[m[1]] = struct{}{}
			}
		}
	}

	expectedKind := derivePromptTemplateKindFromBlocks(tpl.Blocks)
	if tpl.Kind != expectedKind {
		return fmt.Errorf("kind mismatched. got: %q, computed: %q", tpl.Kind, expectedKind)
	}

	// Validate variables.
	allowedVarTypes := map[spec.VarType]struct{}{
		spec.VarString:  {},
		spec.VarNumber:  {},
		spec.VarBoolean: {},
		spec.VarEnum:    {},
		spec.VarDate:    {},
	}
	allowedVarSources := map[spec.VarSource]struct{}{
		spec.SourceUser:   {},
		spec.SourceStatic: {},
	}

	varNames := map[string]spec.VarSource{}
	varDefs := map[string]spec.PromptVariable{}
	for i, v := range tpl.Variables {
		v.Name = strings.TrimSpace(v.Name)
		if err := bundleitemutils.ValidateTag(v.Name); err != nil {
			return fmt.Errorf("variables[%d]: invalid name %q, err %w", i, v.Name, err)
		}

		if _, dup := varNames[v.Name]; dup {
			return fmt.Errorf("duplicate variable %q", v.Name)
		}
		if _, ok := allowedVarTypes[v.Type]; !ok {
			return fmt.Errorf("variables[%d]: invalid type %q", i, v.Type)
		}
		if _, ok := allowedVarSources[v.Source]; !ok {
			return fmt.Errorf("variables[%d]: invalid source %q", i, v.Source)
		}

		// Source-specific requirements.
		switch v.Source {
		case spec.SourceStatic:
			if strings.TrimSpace(v.StaticVal) == "" {
				return fmt.Errorf("variables[%d]: source 'static' requires staticVal", i)
			}
			if v.Required {
				return fmt.Errorf("variables[%d]: static variables cannot be required", i)
			}
		case spec.SourceUser:
			// No validations needed.
		}

		// Type-specific requirements.
		if v.Type == spec.VarEnum {
			if len(v.EnumValues) == 0 {
				return fmt.Errorf("variables[%d]: enum type requires enumValues", i)
			}
			seenEnum := map[string]struct{}{}
			for j, ev := range v.EnumValues {
				ev = strings.TrimSpace(ev)
				if ev == "" {
					return fmt.Errorf("variables[%d]: enumValues[%d] is empty", i, j)
				}
				if _, dup := seenEnum[ev]; dup {
					return fmt.Errorf("variables[%d]: duplicate enum value %q", i, ev)
				}
				seenEnum[ev] = struct{}{}
			}
			if v.Default != nil {
				found := slices.Contains(v.EnumValues, *v.Default)
				if !found {
					return fmt.Errorf("variables[%d]: default %q not in enumValues", i, *v.Default)
				}
			}
		}

		varNames[v.Name] = v.Source
		varDefs[v.Name] = v
	}

	// All placeholders must be declared.
	for ph := range placeholders {
		if _, ok := varNames[ph]; !ok {
			return fmt.Errorf("variable %q used in blocks but not declared", ph)
		}
	}

	// Detect unused variables.
	for n, src := range varNames {
		if _, used := placeholders[n]; used {
			continue
		}
		if src == spec.SourceStatic {
			// Static variables may legitimately stay unused in blocks.
			continue
		}
		return fmt.Errorf("variable %q declared but never used", n)
	}

	isResolved := true
	for ph := range placeholders {
		if !isVariableResolvedLocally(varDefs[ph]) {
			isResolved = false
			break
		}
	}
	if tpl.IsResolved != isResolved {
		return fmt.Errorf("isResolved mismatched. got: %v, computed: %v", tpl.IsResolved, isResolved)
	}

	if err := bundleitemutils.ValidateTags(tpl.Tags); err != nil {
		return err
	}
	return nil
}

func derivePromptTemplateKindFromBlocks(blocks []spec.MessageBlock) spec.PromptTemplateKind {
	if len(blocks) > 0 {
		instructionsOnly := true
		for _, block := range blocks {
			if block.Role != spec.System && block.Role != spec.Developer {
				instructionsOnly = false
				break
			}
		}
		if instructionsOnly {
			return spec.PromptTemplateKindInstructionsOnly
		}
	}

	return spec.PromptTemplateKindGeneric
}

func allowedRolesForKind(kind spec.PromptTemplateKind) map[spec.PromptRoleEnum]struct{} {
	switch kind {
	case spec.PromptTemplateKindInstructionsOnly:
		return map[spec.PromptRoleEnum]struct{}{
			spec.System:    {},
			spec.Developer: {},
		}

	default:
		return map[spec.PromptRoleEnum]struct{}{
			spec.System:    {},
			spec.Developer: {},
			spec.User:      {},
			spec.Assistant: {},
		}
	}
}

func isVariableResolvedLocally(v spec.PromptVariable) bool {
	switch v.Source {
	case spec.SourceStatic:
		return strings.TrimSpace(v.StaticVal) != ""
	case spec.SourceUser:
		if v.Default != nil {
			return true
		}
	default:
		return false
	}
	return false
}

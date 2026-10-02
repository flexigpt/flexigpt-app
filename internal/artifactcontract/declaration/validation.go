package declaration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func ValidateOptionalContent(
	value *string,
) error {
	if value == nil {
		return nil
	}
	if !utf8.ValidString(*value) {
		return fmt.Errorf(
			"%w: inline content must be valid UTF-8",
			spec.ErrInvalid,
		)
	}
	if len(*value) > spec.MaxDefinitionBodyBytes {
		return fmt.Errorf(
			"%w: inline content exceeds %d bytes",
			spec.ErrInvalid,
			spec.MaxDefinitionBodyBytes,
		)
	}
	return nil
}

func ValidateOptionalMediaType(
	value string,
) error {
	if value == "" {
		return nil
	}
	if err := spec.ValidateRequiredText(
		"media type",
		value,
		256,
	); err != nil {
		return err
	}
	return nil
}

// ValidateDeclarationLocatorExclusivity prevents a declaration-source locator
// from being combined with an inline body. Composite declaration locators are
// references to another declaration, not overlays on an inline declaration.
func ValidateDeclarationLocatorExclusivity(
	label string,
	locator *Locator,
	hasInlineBody bool,
) error {
	if locator != nil && hasInlineBody {
		return fmt.Errorf(
			"%w: %s cannot combine a declaration locator with inline fields",
			spec.ErrInvalid,
			label,
		)
	}
	return nil
}

func ValidateJSONValue(
	label string,
	value json.RawMessage,
	maximum int,
) error {
	_, err := canonicalJSONValue(label, value, maximum)
	return err
}

func canonicalJSONValue(
	label string,
	value json.RawMessage,
	maximum int,
) ([]byte, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf(
			"%w: %s is required",
			spec.ErrInvalid,
			label,
		)
	}
	canonical, err := jsonutil.Canonicalize(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if len(canonical) > maximum {
		return nil, fmt.Errorf(
			"%w: %s exceeds %d bytes",
			spec.ErrInvalid,
			label,
			maximum,
		)
	}
	return canonical, nil
}

func ValidateJSONSchemaValue(
	label string,
	value json.RawMessage,
) error {
	canonical, err := canonicalJSONValue(
		label, value, spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return err
	}
	if bytes.Equal(canonical, []byte("true")) ||
		bytes.Equal(canonical, []byte("false")) {
		return nil
	}
	if len(canonical) == 0 || canonical[0] != '{' {
		return fmt.Errorf(
			"%w: %s must be a JSON object or boolean",
			spec.ErrInvalid,
			label,
		)
	}
	if _, err := jsonutil.CompileJSONSchema(canonical); err != nil {
		return fmt.Errorf(
			"%w: %s is not a valid JSON Schema: %w",
			spec.ErrInvalid,
			label,
			err,
		)
	}
	return nil
}

func ValidateOutputMatch(
	label string,
	value OutputMatch,
) error {
	if value.Pointer != "" {
		if err := ValidateJSONPointer(value.Pointer); err != nil {
			return fmt.Errorf("%s pointer: %w", label, err)
		}
	}
	return ValidateJSONSchemaValue(label+" schema", value.Schema)
}

func ValidateJSONPointer(value string) error {
	if value == "" {
		return nil
	}
	if len(value) > spec.MaxURIBytes ||
		!utf8.ValidString(value) ||
		!strings.HasPrefix(value, "/") {
		return fmt.Errorf(
			"%w: invalid RFC 6901 JSON Pointer",
			spec.ErrInvalid,
		)
	}
	for index := 0; index < len(value); index++ {
		if value[index] != '~' {
			continue
		}
		if index+1 >= len(value) ||
			(value[index+1] != '0' &&
				value[index+1] != '1') {
			return fmt.Errorf(
				"%w: invalid RFC 6901 JSON Pointer escape",
				spec.ErrInvalid,
			)
		}
		index++
	}
	return nil
}

func ValidateTextSlice(
	label string,
	values []string,
	maximumBytes int,
) error {
	if len(values) > spec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: %s exceed %d entries",
			spec.ErrInvalid,
			label,
			spec.MaxDefinitionDependencies,
		)
	}
	for index, value := range values {
		if err := spec.ValidateRequiredText(
			label,
			value,
			maximumBytes,
		); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
	}
	return nil
}

func ValidateStringMap(
	label string,
	values map[string]string,
) error {
	if len(values) > spec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: %s exceed %d entries",
			spec.ErrInvalid,
			label,
			spec.MaxDefinitionDependencies,
		)
	}
	for key, value := range values {
		if err := spec.ValidateRequiredText(
			label+" key",
			key,
			spec.MaxURIBytes,
		); err != nil {
			return err
		}
		if len(value) > spec.MaxURIBytes ||
			!utf8.ValidString(value) {
			return fmt.Errorf(
				"%w: %s value for %q is invalid",
				spec.ErrInvalid,
				label,
				key,
			)
		}
	}
	return nil
}

func ValidateRawMessageMap(
	label string,
	values map[string]json.RawMessage,
) error {
	if len(values) > spec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: %s exceed %d entries",
			spec.ErrInvalid,
			label,
			spec.MaxDefinitionDependencies,
		)
	}
	for key, value := range values {
		if err := spec.ValidateRequiredText(
			label+" key",
			key,
			spec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := ValidateJSONValue(
			label+" value",
			value,
			spec.MaxLocalDataBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

func ValidateWorkflowID(
	label string,
	value string,
) error {
	if err := spec.ValidateRequiredText(
		label,
		value,
		spec.MaxLogicalNameBytes,
	); err != nil {
		return err
	}
	return nil
}

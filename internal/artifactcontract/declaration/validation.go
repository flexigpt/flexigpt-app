package declaration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
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
			basespec.ErrInvalid,
		)
	}
	if len(*value) > basespec.MaxDefinitionBodyBytes {
		return fmt.Errorf(
			"%w: inline content exceeds %d bytes",
			basespec.ErrInvalid,
			basespec.MaxDefinitionBodyBytes,
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
	if err := basespec.ValidateRequiredText(
		"media type",
		value,
		256,
	); err != nil {
		return err
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return fmt.Errorf(
			"%w: media type cannot contain whitespace",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func ValidateJSONValue(
	label string,
	value json.RawMessage,
	maximum int,
) error {
	if len(value) == 0 {
		return fmt.Errorf(
			"%w: %s is required",
			basespec.ErrInvalid,
			label,
		)
	}
	canonical, err := jsonutil.Canonicalize(value)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if len(canonical) > maximum {
		return fmt.Errorf(
			"%w: %s exceeds %d bytes",
			basespec.ErrInvalid,
			label,
			maximum,
		)
	}
	return nil
}

func ValidateJSONSchemaValue(
	label string,
	value json.RawMessage,
) error {
	if err := ValidateJSONValue(
		label,
		value,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return err
	}
	canonical, err := jsonutil.Canonicalize(value)
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
			basespec.ErrInvalid,
			label,
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
	if len(value) > basespec.MaxURIBytes ||
		!utf8.ValidString(value) ||
		!strings.HasPrefix(value, "/") {
		return fmt.Errorf(
			"%w: invalid RFC 6901 JSON Pointer",
			basespec.ErrInvalid,
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
				basespec.ErrInvalid,
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
	if len(values) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: %s exceed %d entries",
			basespec.ErrInvalid,
			label,
			basespec.MaxDefinitionDependencies,
		)
	}
	for index, value := range values {
		if err := basespec.ValidateRequiredText(
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
	if len(values) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: %s exceed %d entries",
			basespec.ErrInvalid,
			label,
			basespec.MaxDefinitionDependencies,
		)
	}
	for key, value := range values {
		if err := basespec.ValidateRequiredText(
			label+" key",
			key,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
		if len(value) > basespec.MaxURIBytes ||
			!utf8.ValidString(value) {
			return fmt.Errorf(
				"%w: %s value for %q is invalid",
				basespec.ErrInvalid,
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
	if len(values) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: %s exceed %d entries",
			basespec.ErrInvalid,
			label,
			basespec.MaxDefinitionDependencies,
		)
	}
	for key, value := range values {
		if err := basespec.ValidateRequiredText(
			label+" key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := ValidateJSONValue(
			label+" value",
			value,
			basespec.MaxLocalDataBytes,
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
	if err := basespec.ValidateRequiredText(
		label,
		value,
		basespec.MaxLogicalNameBytes,
	); err != nil {
		return err
	}
	for _, character := range value {
		if unicode.IsSpace(character) {
			return fmt.Errorf(
				"%w: %s cannot contain whitespace",
				basespec.ErrInvalid,
				label,
			)
		}
	}
	return nil
}

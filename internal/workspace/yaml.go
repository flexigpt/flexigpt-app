package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/goccy/go-yaml"
)

const yamlSeparator = "---"

// DecodeYAML converts the restricted Workspace YAML profile to canonical JSON.
// Workspace source files are data documents, not executable YAML. Anchors,
// aliases, tags, and multiple documents are rejected before YAML decoding.
func DecodeYAML(ctx context.Context, content []byte) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(content) == 0 || len(content) > artifactstoreSpec.MaxDefinitionJSONBytes {
		return nil, fmt.Errorf(
			"YAML document must contain between 1 and %d bytes",
			artifactstoreSpec.MaxDefinitionJSONBytes,
		)
	}
	if !utf8.Valid(content) {
		return nil, errors.New("YAML document must be valid UTF-8")
	}
	if err := rejectUnsafeYAMLSyntax(string(content)); err != nil {
		return nil, err
	}

	var value any
	if err := yaml.UnmarshalWithOptions(content, &value, yaml.Strict()); err != nil {
		return nil, fmt.Errorf("decode YAML document: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf(
			"YAML document must contain JSON-compatible values with string mapping keys: %w",
			err,
		)
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize decoded YAML: %w", err)
	}
	if len(canonical) == 0 || canonical[0] != '{' {
		return nil, errors.New("YAML document must decode to a JSON object")
	}
	if len(canonical) > artifactstoreSpec.MaxDefinitionJSONBytes {
		return nil, fmt.Errorf(
			"decoded YAML exceeds %d bytes",
			artifactstoreSpec.MaxDefinitionJSONBytes,
		)
	}
	return json.RawMessage(canonical), nil
}

func rejectUnsafeYAMLSyntax(content string) error {
	seenContent := false
	documentStarted := false
	documentEnded := false

	for lineNumber, line := range strings.Split(content, "\n") {
		code, err := yamlCodeWithoutCommentsOrUnsafeNodes(line)
		if err != nil {
			return fmt.Errorf("YAML line %d: %w", lineNumber+1, err)
		}
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}
		if trimmed == yamlSeparator {
			if documentStarted || seenContent || documentEnded {
				return fmt.Errorf("YAML line %d: multiple YAML documents are not supported", lineNumber+1)
			}
			documentStarted = true
			continue
		}
		if trimmed == "..." {
			if !seenContent || documentEnded {
				return fmt.Errorf("YAML line %d: invalid YAML document terminator", lineNumber+1)
			}
			documentEnded = true
			continue
		}
		if documentEnded {
			return fmt.Errorf("YAML line %d: content after YAML document terminator", lineNumber+1)
		}
		seenContent = true
	}
	return nil
}

func yamlCodeWithoutCommentsOrUnsafeNodes(line string) (string, error) {
	var code strings.Builder
	var quote byte
	escaped := false

	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			code.WriteByte(character)
			switch quote {
			case '"':
				if escaped {
					escaped = false
					continue
				}
				if character == '\\' {
					escaped = true
					continue
				}
				if character == quote {
					quote = 0
				}
			case '\'':
				if character != quote {
					continue
				}
				if index+1 < len(line) && line[index+1] == quote {
					code.WriteByte(line[index+1])
					index++
					continue
				}
				quote = 0
			}
			continue
		}

		switch character {
		case '#':
			return code.String(), nil
		case '"', '\'':
			quote = character
			code.WriteByte(character)
		case '!', '&', '*':
			return "", errors.New("YAML tags, anchors, and aliases are not supported")
		default:
			code.WriteByte(character)
		}
	}
	return code.String(), nil
}

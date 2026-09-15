package declaration

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const APIVersionV1 = "v1"

type Type string

const (
	TypeInstruction Type = "instruction"
	TypeContext     Type = "context"
	TypeTool        Type = "tool"
	TypeModel       Type = "model"
	TypeSkill       Type = "skill"
	TypeMCP         Type = "mcp"
	TypeCollection  Type = "collection"
	TypeAgent       Type = "agent"
	TypeTeam        Type = "team"
	TypeLoop        Type = "loop"
	TypeWorkflow    Type = "workflow"
	TypeWorkspace   Type = "workspace"
)

// TypeMCPPolicy is a supported contract-domain extension type. It is not
// part of the portable CoreType vocabulary because MCP policy is an optional
// policy domain rather than a universally executable capability.
const TypeMCPPolicy Type = "mcp.policy"

func Types() []Type {
	return []Type{
		TypeInstruction,
		TypeContext,
		TypeTool,
		TypeModel,
		TypeSkill,
		TypeMCP,
		TypeCollection,
		TypeAgent,
		TypeTeam,
		TypeLoop,
		TypeWorkflow,
		TypeWorkspace,
		TypeMCPPolicy,
	}
}

func (t Type) Validate() error {
	switch t {
	case TypeInstruction,
		TypeContext,
		TypeTool,
		TypeModel,
		TypeSkill,
		TypeMCP,
		TypeCollection,
		TypeAgent,
		TypeTeam,
		TypeLoop,
		TypeWorkflow,
		TypeMCPPolicy,
		TypeWorkspace:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported Artifact declaration type %q",
			basespec.ErrInvalid,
			t,
		)
	}
}

type Header struct {
	Schema      string                     `json:"$schema,omitempty"`
	APIVersion  string                     `json:"apiVersion,omitempty"`
	Type        Type                       `json:"type"`
	Name        string                     `json:"name"`
	Description string                     `json:"description,omitempty"`
	Locator     *Locator                   `json:"locator,omitempty"`
	Metadata    map[string]json.RawMessage `json:"metadata,omitempty"`
}

type HeaderValidation struct {
	ExpectedType Type
	APIVersion   string
}

func (h Header) Clone() Header {
	output := h
	if h.Locator != nil {
		value := h.Locator.Clone()
		output.Locator = &value
	}
	output.Metadata = CloneRawMessageMap(h.Metadata)
	return output
}

func (h Header) Validate(
	options HeaderValidation,
) error {
	if err := h.Type.Validate(); err != nil {
		return err
	}
	if options.ExpectedType != "" &&
		h.Type != options.ExpectedType {
		return fmt.Errorf(
			"%w: declaration type is %q, expected %q",
			basespec.ErrInvalid,
			h.Type,
			options.ExpectedType,
		)
	}
	if err := basespec.ValidatePortableName(
		"artifact declaration name",
		h.Name,
	); err != nil {
		return err
	}

	if options.APIVersion != "" &&
		h.APIVersion != "" &&
		h.APIVersion != options.APIVersion {
		return fmt.Errorf(
			"%w: declaration apiVersion %q, expected %q",
			basespec.ErrInvalid,
			h.APIVersion,
			options.APIVersion,
		)
	}
	if h.APIVersion != "" {
		if err := basespec.ValidateRequiredText(
			"artifact declaration apiVersion",
			h.APIVersion,
			basespec.MaxVersionBytes,
		); err != nil {
			return err
		}
	}
	if h.Schema != "" {
		if err := validateSchemaURI(h.Schema); err != nil {
			return err
		}
	}
	if err := basespec.ValidateOptionalText(
		"artifact declaration description",
		h.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if h.Locator != nil {
		if err := h.Locator.Validate(); err != nil {
			return fmt.Errorf("artifact declaration locator: %w", err)
		}
		if options.ExpectedType != "" &&
			h.Locator.Kind == LocatorKindCommand {
			switch options.ExpectedType {
			case TypeTool, TypeMCP:
			default:
				return fmt.Errorf(
					"%w: command Locator is valid only for Tool and MCP declarations",
					basespec.ErrInvalid,
				)
			}
		}
	}
	if len(h.Metadata) > basespec.MaxLabels {
		return fmt.Errorf(
			"%w: artifact declaration metadata exceeds %d entries",
			basespec.ErrInvalid,
			basespec.MaxLabels,
		)
	}
	for key, value := range h.Metadata {
		if err := basespec.ValidateRequiredText(
			"artifact declaration metadata key",
			key,
			basespec.MaxSchemaIDBytes,
		); err != nil {
			return err
		}
		if _, err := jsonutil.Canonicalize(value); err != nil {
			return fmt.Errorf(
				"artifact declaration metadata %q: %w",
				key,
				err,
			)
		}
	}
	return nil
}

type OutputMatch struct {
	Pointer string          `json:"pointer,omitempty"`
	Schema  json.RawMessage `json:"schema"`
}

func (m OutputMatch) Clone() OutputMatch {
	output := m
	output.Schema = append(json.RawMessage(nil), m.Schema...)
	return output
}

func validateSchemaURI(value string) error {
	if err := basespec.ValidateRequiredText(
		"Artifact declaration $schema",
		value,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || !parsed.IsAbs() {
		return fmt.Errorf(
			"%w: Artifact declaration $schema must be an absolute URI",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func CloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func CloneStringMap(
	values map[string]string,
) map[string]string {
	if values == nil {
		return nil
	}
	output := make(map[string]string, len(values))
	maps.Copy(output, values)
	return output
}

func CloneRawMessageMap(
	values map[string]json.RawMessage,
) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	output := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		output[key] = append(json.RawMessage(nil), value...)
	}
	return output
}

func CloneEntries(values []Entry) []Entry {
	if values == nil {
		return nil
	}
	output := make([]Entry, len(values))
	for index, value := range values {
		output[index] = value.Clone()
	}
	return output
}

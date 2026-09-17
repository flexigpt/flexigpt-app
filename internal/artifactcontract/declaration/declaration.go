package declaration

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const SchemaVersionV1 = "v1"

type Type string

const (
	TypeText      Type = "text"
	TypeTool      Type = "tool"
	TypeModel     Type = "model"
	TypeSkill     Type = "skill"
	TypeMCP       Type = "mcp"
	TypeMCPPolicy Type = "mcp.policy"
	TypePlugin    Type = "plugin"
	TypeAgent     Type = "agent"
	TypeTeam      Type = "team"
	TypeLoop      Type = "loop"
	TypeWorkflow  Type = "workflow"
	TypeWorkspace Type = "workspace"
)

type InsertTarget string

const (
	InsertInstructions InsertTarget = "instructions"
	InsertUserMessage  InsertTarget = "user-message"
)

func (i InsertTarget) Validate() error {
	switch i {
	case InsertInstructions, InsertUserMessage:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported Text insertion target %q",
			basespec.ErrInvalid,
			i,
		)
	}
}

func Types() []Type {
	return []Type{
		TypeText,
		TypeModel,
		TypeTool,
		TypeSkill,
		TypeMCP,
		TypeMCPPolicy,
		TypePlugin,
		TypeAgent,
		TypeTeam,
		TypeLoop,
		TypeWorkflow,
		TypeWorkspace,
	}
}

func (t Type) Validate() error {
	switch t {
	case TypeText,
		TypeModel,
		TypeTool,
		TypeSkill,
		TypeMCP,
		TypeMCPPolicy,
		TypePlugin,
		TypeAgent,
		TypeTeam,
		TypeLoop,
		TypeWorkflow,
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

const RetiredMCPRuntimeMetadataKey = "flexigpt.site/mcp-runtime-v1"

type Header struct {
	Type        Type                       `json:"type"`
	Name        string                     `json:"name"`
	DisplayName string                     `json:"displayName,omitempty"`
	Description string                     `json:"description,omitempty"`
	Labels      map[string]string          `json:"labels,omitempty"`
	Locator     *Locator                   `json:"locator,omitempty"`
	Metadata    map[string]json.RawMessage `json:"metadata,omitempty"`
}

type HeaderValidation struct {
	ExpectedType Type
	RequireName  bool
}

func (h Header) Clone() Header {
	output := h
	if h.Locator != nil {
		value := h.Locator.Clone()
		output.Locator = &value
	}
	output.Labels = CloneStringMap(h.Labels)
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
	if h.Name == "" {
		if options.RequireName || options.ExpectedType != "" {
			return fmt.Errorf(
				"%w: artifact declaration name is required",
				basespec.ErrInvalid,
			)
		}
	} else {
		if err := basespec.ValidatePortableName(
			"artifact declaration name",
			h.Name,
		); err != nil {
			return err
		}
	}
	if err := basespec.ValidateOptionalText(
		"artifact declaration display name",
		h.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"artifact declaration description",
		h.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateLabels("headers", h.Labels); err != nil {
		return fmt.Errorf("artifact declaration labels: %w", err)
	}
	if h.Locator != nil {
		if err := h.Locator.Validate(); err != nil {
			return fmt.Errorf("artifact declaration locator: %w", err)
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
		if key == RetiredMCPRuntimeMetadataKey {
			return fmt.Errorf(
				"%w: metadata key %q is retired; use direct typed MCP fields",
				basespec.ErrInvalid,
				key,
			)
		}
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

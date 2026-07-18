package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	assistantpresetSpec "github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

var appLocalDefinitionFields = []string{
	"bundleID",
	"collectionID",
	"createdAt",
	"dataSchemaID",
	"id",
	"isBuiltIn",
	"isEnabled",
	"location",
	"modifiedAt",
	"recordID",
	"rootID",
	"schemaVersion",
	"softDeletedAt",
	"sourceID",
}

var modelLocalFields = []string{
	"capabilityProfile",
	"createdAt",
	"defaultModelPresetID",
	"id",
	"isBuiltIn",
	"isEnabled",
	"modelPresets",
	"modifiedAt",
	"schemaVersion",
	"slug",
}

func classifyCurrentDomainDocument(content []byte) (nativeDocumentClass, bool) {
	var object map[string]json.RawMessage
	if json.Unmarshal(content, &object) != nil {
		return nativeDocumentClass{}, false
	}
	schemaVersion := stringField(object, "schemaVersion")
	switch schemaVersion {
	case toolSpec.SchemaVersion:
		if _, hasType := object["type"]; hasType {
			if _, hasSchema := object["argSchema"]; hasSchema {
				return nativeDocumentClass{Kind: KindToolDefinition, Format: formatJSON}, true
			}
		}
	case mcpSpec.MCPSchemaVersion:
		if _, hasTransport := object["transport"]; hasTransport {
			return nativeDocumentClass{Kind: KindMCPServerDefinition, Format: formatJSON}, true
		}
	case assistantpresetSpec.SchemaVersion:
		if _, hasSlug := object["slug"]; hasSlug {
			if _, hasVersion := object["version"]; hasVersion {
				return nativeDocumentClass{Kind: KindAgentDefinition, Format: formatJSON}, true
			}
		}
	}
	return nativeDocumentClass{}, false
}

func normalizeNativeStructuredDocument(
	kind artifactstoreSpec.ArtifactKind,
	object map[string]json.RawMessage,
) (json.RawMessage, map[string]json.RawMessage, error) {
	normalized := cloneRawObject(object)

	if kind == KindAgentDefinition {
		if err := convertAssistantReferences(normalized); err != nil {
			return nil, nil, err
		}
	}

	switch kind {
	case KindAgentDefinition, KindToolDefinition, KindMCPServerDefinition:
		removeRawFields(normalized, appLocalDefinitionFields)
	case KindModelDefinition:
		removeRawFields(normalized, appLocalDefinitionFields)
		if err := stripNestedFields(normalized, "provider", modelLocalFields); err != nil {
			return nil, nil, err
		}
		if err := stripNestedFields(normalized, "model", modelLocalFields); err != nil {
			return nil, nil, err
		}
	}

	if kind == KindMCPServerDefinition {
		if err := stripNestedFields(normalized, "stdio", []string{"secretEnvRefs"}); err != nil {
			return nil, nil, err
		}
		if err := stripNestedFields(
			normalized,
			"streamableHttp",
			[]string{"clientCredentialRef", "secretHeaderRefs"},
		); err != nil {
			return nil, nil, err
		}
	}

	raw, err := json.Marshal(normalized)
	if err != nil {
		return nil, nil, err
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return nil, nil, err
	}
	var canonicalObject map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &canonicalObject); err != nil {
		return nil, nil, err
	}
	return json.RawMessage(canonical), canonicalObject, nil
}

func convertAssistantReferences(object map[string]json.RawMessage) error {
	if raw, exists := object["startingModelPresetRef"]; exists && !isJSONNull(raw) {
		var ref modelpresetSpec.ModelPresetRef
		if err := json.Unmarshal(raw, &ref); err != nil {
			return fmt.Errorf("decode startingModelPresetRef: %w", err)
		}
		if ref.IsZero() {
			return errors.New("startingModelPresetRef is incomplete")
		}
		selector := artifactstoreSpec.ArtifactSelector{
			Kind:        KindModelDefinition,
			LogicalName: artifactstoreSpec.LogicalName(ref.ModelPresetID),
			Labels: map[string]string{
				"provider": string(ref.ProviderName),
			},
		}
		if err := setConvertedField(object, "startingModel", selector); err != nil {
			return err
		}
	}
	delete(object, "startingModelPresetRef")

	if raw, exists := object["startingToolSelections"]; exists && !isJSONNull(raw) {
		var selections []toolSpec.ToolSelection
		if err := json.Unmarshal(raw, &selections); err != nil {
			return fmt.Errorf("decode startingToolSelections: %w", err)
		}
		portable := make([]AgentToolSelection, 0, len(selections))
		for index, selection := range selections {
			if selection.ToolRef.ToolSlug == "" || selection.ToolRef.ToolVersion == "" {
				return fmt.Errorf("startingToolSelections[%d].toolRef is incomplete", index)
			}
			portable = append(portable, AgentToolSelection{
				Selector: artifactstoreSpec.ArtifactSelector{
					Kind:              KindToolDefinition,
					LogicalName:       artifactstoreSpec.LogicalName(selection.ToolRef.ToolSlug),
					VersionConstraint: string(selection.ToolRef.ToolVersion),
				},
				ToolChoicePatch: selection.ToolChoicePatch,
			})
		}
		if err := setConvertedField(object, "startingTools", portable); err != nil {
			return err
		}
	}
	delete(object, "startingToolSelections")

	if raw, exists := object["startingSkillSelections"]; exists && !isJSONNull(raw) {
		var selections []struct {
			SkillRef struct {
				SkillSlug string `json:"skillSlug"`
			} `json:"skillRef"`
			PreLoadAsActive   bool `json:"preLoadAsActive"`
			UseAsInstructions bool `json:"useAsInstructions"`
		}
		if err := json.Unmarshal(raw, &selections); err != nil {
			return fmt.Errorf("decode startingSkillSelections: %w", err)
		}
		portable := make([]AgentSkillSelection, 0, len(selections))
		for index, selection := range selections {
			if strings.TrimSpace(selection.SkillRef.SkillSlug) == "" {
				return fmt.Errorf("startingSkillSelections[%d].skillRef is incomplete", index)
			}
			portable = append(portable, AgentSkillSelection{
				Selector: artifactstoreSpec.ArtifactSelector{
					Kind:        KindSkillDefinition,
					LogicalName: artifactstoreSpec.LogicalName(selection.SkillRef.SkillSlug),
				},
				PreLoadAsActive:   selection.PreLoadAsActive,
				UseAsInstructions: selection.UseAsInstructions,
			})
		}
		if err := setConvertedField(object, "startingSkills", portable); err != nil {
			return err
		}
	}
	delete(object, "startingSkillSelections")

	if raw, exists := object["startingMCPContext"]; exists && !isJSONNull(raw) {
		var current mcpSpec.MCPConversationContext
		if err := json.Unmarshal(raw, &current); err != nil {
			return fmt.Errorf("decode startingMCPContext: %w", err)
		}
		if len(current.Resources) != 0 ||
			len(current.ResourceTemplates) != 0 ||
			len(current.Prompts) != 0 {
			return errors.New(
				"startingMCPContext discovery resources, templates, and prompts are runtime-local and cannot be portable",
			)
		}
		portable := make([]AgentMCPServerSelection, 0, len(current.Servers))
		for serverIndex, server := range current.Servers {
			if server.ServerID == "" {
				return fmt.Errorf("startingMCPContext.servers[%d].serverID is empty", serverIndex)
			}
			converted := AgentMCPServerSelection{
				Selector: artifactstoreSpec.ArtifactSelector{
					Kind:        KindMCPServerDefinition,
					LogicalName: artifactstoreSpec.LogicalName(server.ServerID),
				},
				ToolExposure:              server.ToolExposure,
				IncludeServerInstructions: server.IncludeServerInstructions,
			}
			for _, tool := range server.SelectedTools {
				converted.SelectedTools = append(converted.SelectedTools, AgentMCPToolSelection{
					ToolName:       tool.ToolName,
					ApprovalRule:   tool.ApprovalRule,
					ExecutionMode:  tool.ExecutionMode,
					AppResourceURI: tool.AppResourceURI,
					Visibility:     append([]string(nil), tool.Visibility...),
				})
			}
			portable = append(portable, converted)
		}
		if err := setConvertedField(object, "startingMCPServers", portable); err != nil {
			return err
		}
	}
	delete(object, "startingMCPContext")
	return nil
}

func setConvertedField(object map[string]json.RawMessage, field string, value any) error {
	if existing, exists := object[field]; exists && !isJSONNull(existing) {
		return fmt.Errorf("both app-local and portable %s fields are present", field)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	object[field] = raw
	return nil
}

func nativeLogicalName(
	kind artifactstoreSpec.ArtifactKind,
	object map[string]json.RawMessage,
	locator artifactstoreSpec.SourceLocator,
) string {
	switch kind {
	case KindAgentDefinition, KindToolDefinition:
		if value := stringField(object, "slug", "name", "id"); value != "" {
			return value
		}
	case KindMCPServerDefinition:
		if value := stringField(object, "id", "name"); value != "" {
			return value
		}
	case KindModelDefinition:
		if raw, exists := object["model"]; exists {
			var model map[string]json.RawMessage
			if json.Unmarshal(raw, &model) == nil {
				if value := stringField(model, "id", "slug", "name"); value != "" {
					return value
				}
			}
		}
	default:
		if value := stringField(object, "name", "slug", "id"); value != "" {
			return value
		}
	}
	return sourceLogicalName(locator)
}

func nativeModelProviderName(object map[string]json.RawMessage) string {
	if value := stringField(object, "providerName"); value != "" {
		return value
	}
	raw, exists := object["provider"]
	if !exists {
		return ""
	}
	var provider map[string]json.RawMessage
	if json.Unmarshal(raw, &provider) != nil {
		return ""
	}
	return stringField(provider, "name")
}

func stripNestedFields(
	object map[string]json.RawMessage,
	field string,
	fields []string,
) error {
	raw, exists := object[field]
	if !exists || isJSONNull(raw) {
		return nil
	}
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err != nil {
		return fmt.Errorf("%s must be an object: %w", field, err)
	}
	removeRawFields(nested, fields)
	encoded, err := json.Marshal(nested)
	if err != nil {
		return err
	}
	object[field] = encoded
	return nil
}

func removeRawFields(object map[string]json.RawMessage, fields []string) {
	for _, field := range fields {
		delete(object, field)
	}
}

func cloneRawObject(input map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(input))
	for key, value := range input {
		out[key] = append(json.RawMessage(nil), value...)
	}
	return out
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

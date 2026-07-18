package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
	assistantpresetSpec "github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	assistantpresetStore "github.com/flexigpt/flexigpt-app/internal/assistantpreset/store"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/flexigpt/flexigpt-app/internal/tool/storehelper"
)

const (
	defaultProjectedVersion          = "1"
	projectionValidationRootID       = artifactstoreSpec.RootID("00000000-0000-7000-8000-000000000001")
	projectionValidationSourceID     = artifactstoreSpec.SourceID("00000000-0000-7000-8000-000000000002")
	projectionValidationCollectionID = artifactstoreSpec.CollectionID("00000000-0000-7000-8000-000000000003")
	projectionValidationRecordID     = artifactstoreSpec.RecordID("00000000-0000-7000-8000-000000000004")
)

type toolProjector struct{}

func (toolProjector) Kind() artifactstoreSpec.ArtifactKind {
	return KindToolDefinition
}

func (toolProjector) Project(
	_ context.Context,
	input ProjectionInput,
) (any, []artifactstoreSpec.Diagnostic) {
	projected, err := projectToolDefinition(input)
	if err != nil {
		return nil, projectorDiagnostics(input, err)
	}
	return projected, nil
}

type mcpProjector struct{}

func (mcpProjector) Kind() artifactstoreSpec.ArtifactKind {
	return KindMCPServerDefinition
}

func (mcpProjector) Project(
	_ context.Context,
	input ProjectionInput,
) (any, []artifactstoreSpec.Diagnostic) {
	projected, err := projectMCPDefinition(input)
	if err != nil {
		return nil, projectorDiagnostics(input, err)
	}
	return projected, nil
}

type modelProjector struct{}

func (modelProjector) Kind() artifactstoreSpec.ArtifactKind {
	return KindModelDefinition
}

func (modelProjector) Project(
	_ context.Context,
	input ProjectionInput,
) (any, []artifactstoreSpec.Diagnostic) {
	projected, err := projectModelDefinition(input)
	if err != nil {
		return nil, projectorDiagnostics(input, err)
	}
	return projected, nil
}

type agentProjector struct{}

func (agentProjector) Kind() artifactstoreSpec.ArtifactKind {
	return KindAgentDefinition
}

func (agentProjector) Project(
	_ context.Context,
	input ProjectionInput,
) (any, []artifactstoreSpec.Diagnostic) {
	projected, err := projectAgentDefinition(input)
	if err != nil {
		return nil, projectorDiagnostics(input, err)
	}
	return projected, nil
}

func validateDomainDefinition(definition artifactstoreSpec.CanonicalDefinition) error {
	if err := rejectAppLocalDefinitionData(definition); err != nil {
		return err
	}
	input := validationProjectionInput(definition)
	switch definition.Kind {
	case KindSkillDefinition:
		projected, err := parseSkillDefinition(definition)
		if err != nil {
			return err
		}
		return applyDomainSkillProjection(input, &projected)
	case KindToolDefinition:
		_, err := projectToolDefinition(input)
		return err
	case KindMCPServerDefinition:
		_, err := projectMCPDefinition(input)
		return err
	case KindModelDefinition:
		_, err := projectModelDefinition(input)
		return err
	case KindAgentDefinition:
		_, err := projectAgentDefinition(input)
		return err
	default:
		return nil
	}
}

func applyDomainSkillProjection(input ProjectionInput, projected *ProjectedSkill) error {
	if projected == nil {
		return errors.New("projected skill is nil")
	}
	bundleID, err := projectionBundleID(input.Record)
	if err != nil {
		return err
	}

	arguments := make([]skillSpec.SkillArgument, 0, len(projected.Arguments))
	for _, argument := range projected.Arguments {
		arguments = append(arguments, skillSpec.SkillArgument{
			Name:        argument.Name,
			Description: argument.Description,
			Default:     argument.Default,
		})
	}
	if err := skillStore.ValidateSkillArtifactMetadata(
		projected.Name,
		projected.Description,
		arguments,
	); err != nil {
		return err
	}

	var rawFrontmatter map[string]any
	if err := json.Unmarshal(projected.Frontmatter, &rawFrontmatter); err != nil {
		return fmt.Errorf("decode projected skill frontmatter: %w", err)
	}
	var metadata struct {
		Tags []string `json:"tags,omitempty"`
	}
	if err := json.Unmarshal(projected.Frontmatter, &metadata); err != nil {
		return fmt.Errorf("decode projected skill tags: %w", err)
	}

	resources := skillSpec.SkillResourceInfo{}
	if len(input.Definition.AssetManifest) > 0 {
		resources.HasResources = true
		resources.TotalCount = len(input.Definition.AssetManifest)
		maximum := min(len(input.Definition.AssetManifest), skillSpec.MaxSkillResourceLocations)
		resources.Locations = make([]string, 0, maximum)
		for index, asset := range input.Definition.AssetManifest {
			if index >= maximum {
				break
			}
			resources.Locations = append(resources.Locations, string(asset.Path))
		}
		resources.MoreLocations = len(input.Definition.AssetManifest) > maximum
	}

	isBuiltIn := projectionIsBuiltIn(input)
	skillType := skillSpec.SkillTypeFS
	if isBuiltIn {
		skillType = skillSpec.SkillTypeEmbeddedFS
	}
	location := path.Dir(string(input.Record.Locator))
	if location == "" {
		location = "."
	}
	skill := skillSpec.Skill{
		SchemaVersion:  skillSpec.SkillSchemaVersion,
		ID:             bundleitemutils.ItemID(input.Record.RecordID),
		Slug:           bundleitemutils.ItemSlug(input.Record.Name),
		Type:           skillType,
		Location:       location,
		Name:           projected.Name,
		DisplayName:    projected.DisplayName,
		Description:    projected.Description,
		Tags:           append([]string(nil), metadata.Tags...),
		Insert:         skillSpec.SkillInsert(projected.Insert),
		Arguments:      arguments,
		Resources:      resources,
		RawFrontmatter: rawFrontmatter,
		Digest:         string(input.Definition.Digest),
		IsEnabled:      input.Record.Enabled,
		IsBuiltIn:      isBuiltIn,
		CreatedAt:      input.Record.CreatedAt,
		ModifiedAt:     input.Record.ModifiedAt,
	}
	if err := skillStore.ValidateSkill(&skill); err != nil {
		return fmt.Errorf("validate projected Skill: %w", err)
	}
	projected.Skill = skill
	projected.SkillRef = skillSpec.SkillRef{
		BundleID:  bundleID,
		SkillSlug: skill.Slug,
		SkillID:   skill.ID,
	}
	return nil
}

func projectToolDefinition(input ProjectionInput) (ProjectedTool, error) {
	bundleID, err := projectionBundleID(input.Record)
	if err != nil {
		return ProjectedTool{}, err
	}
	var tool toolSpec.Tool
	if err := decodeStrictJSONObject(input.Definition.DefinitionJSON, &tool, false); err != nil {
		return ProjectedTool{}, fmt.Errorf("decode tool definition: %w", err)
	}
	version := string(input.Record.Version)
	if version == "" {
		version = defaultProjectedVersion
	}
	tool.SchemaVersion = toolSpec.SchemaVersion
	tool.ID = bundleitemutils.ItemID(input.Record.RecordID)
	tool.Slug = bundleitemutils.ItemSlug(input.Record.Name)
	tool.Version = bundleitemutils.ItemVersion(version)
	if strings.TrimSpace(tool.DisplayName) == "" {
		tool.DisplayName = projectionDisplayName(input)
	}
	tool.IsEnabled = input.Record.Enabled
	tool.IsBuiltIn = projectionIsBuiltIn(input)
	tool.CreatedAt = input.Record.CreatedAt
	tool.ModifiedAt = input.Record.ModifiedAt
	if err := storehelper.ValidateTool(&tool); err != nil {
		return ProjectedTool{}, fmt.Errorf("validate projected Tool: %w", err)
	}
	return ProjectedTool{
		Tool: tool,
		ToolRef: toolSpec.ToolRef{
			BundleID:    bundleID,
			ToolSlug:    tool.Slug,
			ToolVersion: tool.Version,
		},
	}, nil
}

func projectMCPDefinition(input ProjectionInput) (ProjectedMCPServer, error) {
	bundleID, err := projectionBundleID(input.Record)
	if err != nil {
		return ProjectedMCPServer{}, err
	}
	var config mcpSpec.MCPServerConfig
	if err := decodeStrictJSONObject(input.Definition.DefinitionJSON, &config, false); err != nil {
		return ProjectedMCPServer{}, fmt.Errorf("decode MCP server definition: %w", err)
	}
	if config.Transport == "" {
		var common struct {
			Type    string            `json:"type,omitempty"`
			Command string            `json:"command,omitempty"`
			Args    []string          `json:"args,omitempty"`
			Env     map[string]string `json:"env,omitempty"`
			Cwd     string            `json:"cwd,omitempty"`
			URL     string            `json:"url,omitempty"`
			Headers map[string]string `json:"headers,omitempty"`
		}
		if err := json.Unmarshal(input.Definition.DefinitionJSON, &common); err != nil {
			return ProjectedMCPServer{}, err
		}
		switch {
		case common.Command != "" && common.URL != "":
			return ProjectedMCPServer{}, errors.New("MCP server cannot declare both command and url")
		case common.Command != "":
			config.Transport = mcpSpec.MCPTransportStdio
			config.Stdio = &mcpSpec.MCPStdioConfig{
				Command:    common.Command,
				Args:       append([]string(nil), common.Args...),
				WorkingDir: common.Cwd,
				Env:        common.Env,
			}
		case common.URL != "":
			config.Transport = mcpSpec.MCPTransportStreamableHTTP
			config.StreamableHTTP = &mcpSpec.MCPStreamableHTTPConfig{
				URL:      common.URL,
				AuthMode: mcpSpec.MCPHTTPAuthNone,
				Headers:  common.Headers,
			}
		default:
			return ProjectedMCPServer{}, errors.New("MCP server requires command or url")
		}
	}

	if config.DefaultPolicy.DefaultApprovalRule == "" &&
		config.DefaultPolicy.DefaultExecutionMode == "" {
		config.DefaultPolicy = mcpSpec.DefaultMCPServerPolicy()
	}
	config.SchemaVersion = mcpSpec.MCPSchemaVersion
	config.BundleID = bundleID
	config.ID = mcpSpec.MCPServerID(input.Record.Name)
	if strings.TrimSpace(config.DisplayName) == "" {
		config.DisplayName = projectionDisplayName(input)
	}
	config.Enabled = input.Record.Enabled
	config.IsBuiltIn = projectionIsBuiltIn(input)
	config.CreatedAt = input.Record.CreatedAt
	config.ModifiedAt = input.Record.ModifiedAt
	if err := mcpStore.ValidateServerConfig(&config); err != nil {
		return ProjectedMCPServer{}, fmt.Errorf("validate projected MCP server: %w", err)
	}
	return ProjectedMCPServer{Server: config}, nil
}

func projectModelDefinition(input ProjectionInput) (ProjectedModel, error) {
	var document struct {
		ProviderName inferenceSpec.ProviderName     `json:"providerName,omitempty"`
		Provider     modelpresetSpec.ProviderPreset `json:"provider"`
		Model        modelpresetSpec.ModelPreset    `json:"model"`
	}
	if err := decodeStrictJSONObject(input.Definition.DefinitionJSON, &document, false); err != nil {
		return ProjectedModel{}, fmt.Errorf("decode model definition: %w", err)
	}

	provider := document.Provider
	if provider.Name == "" {
		provider.Name = document.ProviderName
	}
	if provider.Name == "" && input.Definition.Labels != nil {
		provider.Name = inferenceSpec.ProviderName(input.Definition.Labels["provider"])
	}
	if provider.Name == "" {
		return ProjectedModel{}, errors.New("model definition requires provider.name or providerName")
	}

	model := document.Model
	modelID := modelpresetSpec.ModelPresetID(input.Record.Name)
	model.SchemaVersion = modelpresetSpec.SchemaVersion
	model.ID = modelID
	if model.Name == "" {
		model.Name = modelpresetSpec.ModelName(input.Definition.LogicalName)
	}
	if model.Slug == "" {
		model.Slug = modelpresetSpec.ModelSlug(input.Record.Name)
	}
	if model.DisplayName == "" {
		model.DisplayName = modelpresetSpec.ModelDisplayName(projectionDisplayName(input))
	}
	model.IsEnabled = input.Record.Enabled
	model.IsBuiltIn = projectionIsBuiltIn(input)
	model.CreatedAt = input.Record.CreatedAt
	model.ModifiedAt = input.Record.ModifiedAt
	if err := modelpresetStore.ValidateModelPreset(&model); err != nil {
		return ProjectedModel{}, fmt.Errorf("validate projected ModelPreset: %w", err)
	}

	provider.SchemaVersion = modelpresetSpec.SchemaVersion
	if provider.DisplayName == "" {
		provider.DisplayName = modelpresetSpec.ProviderDisplayName(provider.Name)
	}
	provider.IsEnabled = input.Record.Enabled
	provider.IsBuiltIn = projectionIsBuiltIn(input)
	provider.CreatedAt = input.Record.CreatedAt
	provider.ModifiedAt = input.Record.ModifiedAt
	provider.DefaultModelPresetID = modelID
	provider.ModelPresets = map[modelpresetSpec.ModelPresetID]modelpresetSpec.ModelPreset{
		modelID: model,
	}
	if err := modelpresetStore.ValidateProviderPreset(&provider); err != nil {
		return ProjectedModel{}, fmt.Errorf("validate projected ProviderPreset: %w", err)
	}
	return ProjectedModel{
		Provider: provider,
		Model:    model,
		Ref: modelpresetSpec.ModelPresetRef{
			ProviderName:  provider.Name,
			ModelPresetID: model.ID,
		},
	}, nil
}

func projectAgentDefinition(input ProjectionInput) (ProjectedAgent, error) {
	bundleID, err := projectionBundleID(input.Record)
	if err != nil {
		return ProjectedAgent{}, err
	}
	var document AgentDefinitionDocument
	if err := decodeStrictJSONObject(input.Definition.DefinitionJSON, &document, false); err != nil {
		return ProjectedAgent{}, fmt.Errorf("decode agent definition: %w", err)
	}
	if err := validateAgentDocument(document); err != nil {
		return ProjectedAgent{}, err
	}

	version := string(input.Record.Version)
	if version == "" {
		version = defaultProjectedVersion
	}
	preset := assistantpresetSpec.AssistantPreset{
		SchemaVersion:                    assistantpresetSpec.SchemaVersion,
		ID:                               bundleitemutils.ItemID(input.Record.RecordID),
		Slug:                             bundleitemutils.ItemSlug(input.Record.Name),
		Version:                          bundleitemutils.ItemVersion(version),
		DisplayName:                      projectionDisplayName(input),
		Description:                      input.Definition.Description,
		IsEnabled:                        input.Record.Enabled,
		IsBuiltIn:                        projectionIsBuiltIn(input),
		StartingText:                     document.StartingText,
		StartingIncludeModelSystemPrompt: document.StartingIncludeModelSystemPrompt,
		CreatedAt:                        input.Record.CreatedAt,
		ModifiedAt:                       input.Record.ModifiedAt,
	}
	if err := assistantpresetStore.ValidateAssistantPresetStructure(&preset); err != nil {
		return ProjectedAgent{}, fmt.Errorf("validate projected AssistantPreset: %w", err)
	}
	return ProjectedAgent{
		BundleID:           bundleID,
		Preset:             preset,
		StartingModel:      document.StartingModel,
		StartingTools:      append([]AgentToolSelection(nil), document.StartingTools...),
		StartingSkills:     append([]AgentSkillSelection(nil), document.StartingSkills...),
		StartingMCPServers: append([]AgentMCPServerSelection(nil), document.StartingMCPServers...),
		Definition:         append(json.RawMessage(nil), input.Definition.DefinitionJSON...),
	}, nil
}

func validateAgentDocument(document AgentDefinitionDocument) error {
	if document.StartingModel != nil {
		if err := validate.ValidateArtifactSelector(*document.StartingModel); err != nil {
			return fmt.Errorf("startingModel: %w", err)
		}
	}
	for index, selection := range document.StartingTools {
		if err := validate.ValidateArtifactSelector(selection.Selector); err != nil {
			return fmt.Errorf("startingTools[%d].selector: %w", index, err)
		}
	}
	for index, selection := range document.StartingSkills {
		if err := validate.ValidateArtifactSelector(selection.Selector); err != nil {
			return fmt.Errorf("startingSkills[%d].selector: %w", index, err)
		}
		if selection.PreLoadAsActive && selection.UseAsInstructions {
			return fmt.Errorf(
				"startingSkills[%d] cannot set both preLoadAsActive and useAsInstructions",
				index,
			)
		}
	}
	for index, selection := range document.StartingMCPServers {
		if err := validate.ValidateArtifactSelector(selection.Selector); err != nil {
			return fmt.Errorf("startingMCPServers[%d].selector: %w", index, err)
		}
		switch selection.ToolExposure {
		case "", mcpSpec.MCPToolExposureNone, mcpSpec.MCPToolExposureAll:
			if len(selection.SelectedTools) != 0 {
				return fmt.Errorf(
					"startingMCPServers[%d].selectedTools requires toolExposure=%q",
					index,
					mcpSpec.MCPToolExposureSelected,
				)
			}
		case mcpSpec.MCPToolExposureSelected:
			if len(selection.SelectedTools) == 0 {
				return fmt.Errorf("startingMCPServers[%d].selectedTools is empty", index)
			}
		default:
			return fmt.Errorf(
				"startingMCPServers[%d].toolExposure %q is invalid",
				index,
				selection.ToolExposure,
			)
		}
		for toolIndex, tool := range selection.SelectedTools {
			if strings.TrimSpace(tool.ToolName) == "" {
				return fmt.Errorf(
					"startingMCPServers[%d].selectedTools[%d].toolName is empty",
					index,
					toolIndex,
				)
			}
		}
	}
	return nil
}

func projectionBundleID(record artifactstoreSpec.ArtifactRecord) (bundleitemutils.BundleID, error) {
	if record.CollectionID == nil {
		return "", errors.New("domain projection requires a Workspace collection")
	}
	return bundleitemutils.BundleID(*record.CollectionID), nil
}

func projectionIsBuiltIn(input ProjectionInput) bool {
	for _, attachment := range input.Workspace.Attachments {
		if attachment.SourceID == input.Record.SourceID {
			return attachment.Role == RoleBuiltIn
		}
	}
	return false
}

func projectionDisplayName(input ProjectionInput) string {
	if strings.TrimSpace(input.Definition.DisplayName) != "" {
		return input.Definition.DisplayName
	}
	if strings.TrimSpace(string(input.Definition.LogicalName)) != "" {
		return string(input.Definition.LogicalName)
	}
	return string(input.Record.Name)
}

func validationProjectionInput(
	definition artifactstoreSpec.CanonicalDefinition,
) ProjectionInput {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	collectionID := projectionValidationCollectionID
	version := artifactstoreSpec.RecordVersion(definition.LogicalVersion)
	if version == "" {
		version = defaultProjectedVersion
	}
	locator := artifactstoreSpec.SourceLocator("definition.json")
	if definition.Kind == KindSkillDefinition {
		locator = "skill/SKILL.md"
	}
	return ProjectionInput{
		Workspace: Workspace{
			Root: artifactstoreSpec.ArtifactRoot{RootID: projectionValidationRootID},
			Attachments: []artifactstoreSpec.RootSourceAttachment{{
				RootID:   projectionValidationRootID,
				SourceID: projectionValidationSourceID,
				Role:     RoleBuiltIn,
				Enabled:  true,
			}},
		},
		Record: artifactstoreSpec.ArtifactRecord{
			RecordID:     projectionValidationRecordID,
			RootID:       projectionValidationRootID,
			CollectionID: &collectionID,
			Kind:         definition.Kind,
			Name: workspaceRecordName(
				definition.LogicalName,
				projectionValidationSourceID,
				locator,
				"",
			),
			Version:    version,
			SourceID:   projectionValidationSourceID,
			Locator:    locator,
			Enabled:    true,
			CreatedAt:  now,
			ModifiedAt: now,
		},
		Definition: definition,
	}
}

func rejectAppLocalDefinitionData(definition artifactstoreSpec.CanonicalDefinition) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(definition.DefinitionJSON, &object); err != nil {
		return err
	}
	for _, field := range appLocalDefinitionFields {
		if raw, exists := object[field]; exists && !isJSONNull(raw) {
			return fmt.Errorf("portable %s definition must not contain app-local field %q", definition.Kind, field)
		}
	}
	for _, field := range []string{
		"startingMCPContext",
		"startingModelPresetRef",
		"startingSkillSelections",
		"startingToolSelections",
	} {
		if raw, exists := object[field]; exists && !isJSONNull(raw) {
			return fmt.Errorf("portable agent definition must use selectors instead of %q", field)
		}
	}
	if definition.Kind == KindMCPServerDefinition {
		for parent, fields := range map[string][]string{
			"stdio":          {"secretEnvRefs"},
			"streamableHttp": {"clientCredentialRef", "secretHeaderRefs"},
		} {
			raw, exists := object[parent]
			if !exists || isJSONNull(raw) {
				continue
			}
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) != nil {
				continue
			}
			for _, field := range fields {
				if value, exists := nested[field]; exists && !isJSONNull(value) {
					return fmt.Errorf(
						"portable MCP definition must not contain app-local setup reference %q",
						parent+"."+field,
					)
				}
			}
		}
	}
	return nil
}

var (
	_ ResourceProjector = toolProjector{}
	_ ResourceProjector = mcpProjector{}
	_ ResourceProjector = modelProjector{}
	_ ResourceProjector = agentProjector{}
)

package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func validateStartingMCPContextStructure(
	starting *mcpSpec.MCPConversationContext,
) error {
	if starting == nil {
		return nil
	}

	seenServers := make(map[artifact.ArtifactRef]struct{}, len(starting.Servers))
	for index, server := range starting.Servers {
		field := fmt.Sprintf("startingMCPContext.servers[%d]", index)
		if err := validateMCPServerSelectionStructure(field, server); err != nil {
			return err
		}
		if _, duplicate := seenServers[server.Server]; duplicate {
			return fmt.Errorf("%s: duplicate server selection", field)
		}
		seenServers[server.Server] = struct{}{}
	}

	if err := validateMCPResourcesStructure(starting.Resources); err != nil {
		return err
	}
	if err := validateMCPResourceTemplatesStructure(starting.ResourceTemplates); err != nil {
		return err
	}
	return validateMCPPromptsStructure(starting.Prompts)
}

func validateMCPServerSelectionStructure(
	field string,
	server mcpSpec.MCPServerSelection,
) error {
	if err := server.Server.Validate(); err != nil {
		return fmt.Errorf("%s.server: %w", field, err)
	}

	switch server.ToolExposure {
	case "":
		if len(server.SelectedTools) != 0 {
			return fmt.Errorf(
				"%s: toolExposure must be selected when selectedTools is non-empty",
				field,
			)
		}
	case mcpSpec.MCPToolExposureNone, mcpSpec.MCPToolExposureAll:
		if len(server.SelectedTools) != 0 {
			return fmt.Errorf(
				"%s: selectedTools must be empty when toolExposure is %q",
				field,
				server.ToolExposure,
			)
		}
	case mcpSpec.MCPToolExposureSelected:
		if len(server.SelectedTools) == 0 {
			return fmt.Errorf(
				"%s: selectedTools must be non-empty when toolExposure is selected",
				field,
			)
		}
	default:
		return fmt.Errorf("%s: invalid toolExposure %q", field, server.ToolExposure)
	}

	seenTools := make(map[string]struct{}, len(server.SelectedTools))
	for index, tool := range server.SelectedTools {
		toolField := fmt.Sprintf("%s.selectedTools[%d]", field, index)
		if err := validateMCPToolSelectionStructure(toolField, server, tool); err != nil {
			return err
		}
		name := strings.TrimSpace(tool.ToolName)
		if _, duplicate := seenTools[name]; duplicate {
			return fmt.Errorf("%s: duplicate toolName %q", toolField, name)
		}
		seenTools[name] = struct{}{}
	}
	return nil
}

func validateMCPToolSelectionStructure(
	field string,
	parent mcpSpec.MCPServerSelection,
	selection mcpSpec.MCPToolSelection,
) error {
	if err := selection.Server.Validate(); err != nil {
		return fmt.Errorf("%s.server: %w", field, err)
	}
	if selection.Server != parent.Server ||
		strings.TrimSpace(selection.ToolName) == "" {
		return fmt.Errorf(
			"%s: selected tool must identify the parent server and toolName",
			field,
		)
	}
	if err := validateOptionalMCPApprovalRule(
		field+".approvalRule",
		selection.ApprovalRule,
	); err != nil {
		return err
	}
	return validateOptionalMCPExecutionMode(
		field+".executionMode",
		selection.ExecutionMode,
	)
}

func validateMCPResourcesStructure(
	resources []mcpSpec.MCPResourceRef,
) error {
	seen := make(map[string]struct{}, len(resources))
	for index, resource := range resources {
		field := fmt.Sprintf("startingMCPContext.resources[%d]", index)
		if err := resource.Server.Validate(); err != nil {
			return fmt.Errorf("%s.server: %w", field, err)
		}
		if strings.TrimSpace(resource.URI) == "" {
			return fmt.Errorf("%s: uri is required", field)
		}
		key := mcpReferenceKey(resource.Server, resource.URI)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%s: duplicate resource", field)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateMCPResourceTemplatesStructure(
	templates []mcpSpec.MCPResourceTemplateSelection,
) error {
	seen := make(map[string]struct{}, len(templates))
	for index, selection := range templates {
		field := fmt.Sprintf("startingMCPContext.resourceTemplates[%d]", index)
		ref := selection.MCPResourceTemplateRef
		if err := ref.Server.Validate(); err != nil {
			return fmt.Errorf("%s.server: %w", field, err)
		}
		if strings.TrimSpace(ref.URITemplate) == "" {
			return fmt.Errorf("%s: uriTemplate is required", field)
		}
		if err := validateMCPArgumentValuesStructure(
			field+".argumentValues",
			selection.ArgumentValues,
		); err != nil {
			return err
		}
		if err := validateMCPRequiredArguments(
			field+".argumentValues",
			ref.Arguments,
			selection.ArgumentValues,
		); err != nil {
			return err
		}
		key := mcpReferenceKey(ref.Server, ref.URITemplate)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%s: duplicate resource template", field)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateMCPPromptsStructure(
	prompts []mcpSpec.MCPPromptSelection,
) error {
	seen := make(map[string]struct{}, len(prompts))
	for index, selection := range prompts {
		field := fmt.Sprintf("startingMCPContext.prompts[%d]", index)
		if err := selection.Server.Validate(); err != nil {
			return fmt.Errorf("%s.server: %w", field, err)
		}
		if strings.TrimSpace(selection.PromptName) == "" {
			return fmt.Errorf("%s: promptName is required", field)
		}
		if err := validateMCPArgumentValuesStructure(
			field+".argumentValues",
			selection.ArgumentValues,
		); err != nil {
			return err
		}
		if err := validateMCPRequiredArguments(
			field+".argumentValues",
			selection.Arguments,
			selection.ArgumentValues,
		); err != nil {
			return err
		}
		key := mcpReferenceKey(selection.Server, selection.PromptName)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%s: duplicate prompt", field)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateMCPArgumentValuesStructure(
	field string,
	values map[string]string,
) error {
	for name := range values {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%s: argument name is empty", field)
		}
	}
	return nil
}

func validateMCPRequiredArguments(
	field string,
	definitions map[string]mcpSpec.MCPArgumentDefinition,
	values map[string]string,
) error {
	for name, definition := range definitions {
		if !definition.Required {
			continue
		}
		argumentName := strings.TrimSpace(definition.Name)
		if argumentName == "" {
			argumentName = strings.TrimSpace(name)
		}
		if argumentName != "" &&
			strings.TrimSpace(values[argumentName]) == "" {
			return fmt.Errorf("%s: missing required argument %q", field, argumentName)
		}
	}
	return nil
}

func validateOptionalMCPApprovalRule(
	field string,
	rule *mcpSpec.MCPApprovalRule,
) error {
	if rule == nil {
		return nil
	}
	switch *rule {
	case mcpSpec.MCPApprovalRuleAsk,
		mcpSpec.MCPApprovalRuleAllow,
		mcpSpec.MCPApprovalRuleDeny:
		return nil
	default:
		return fmt.Errorf("%s: invalid approvalRule %q", field, *rule)
	}
}

func validateOptionalMCPExecutionMode(
	field string,
	mode *mcpSpec.MCPExecutionMode,
) error {
	if mode == nil {
		return nil
	}
	switch *mode {
	case mcpSpec.MCPExecutionModeManual,
		mcpSpec.MCPExecutionModeAuto:
		return nil
	default:
		return fmt.Errorf("%s: invalid executionMode %q", field, *mode)
	}
}

func validateStartingMCPContextReferences(
	ctx context.Context,
	starting *mcpSpec.MCPConversationContext,
	lookups ReferenceLookups,
) error {
	if starting == nil || isEmptyMCPConversationContext(*starting) {
		return nil
	}
	if lookups.MCPContext == nil {
		return errors.New("mcp context lookup not configured")
	}
	if err := lookups.MCPContext.ValidateMCPConversationContext(ctx, *starting); err != nil {
		return fmt.Errorf("startingMCPContext: %w", err)
	}
	return nil
}

func isEmptyMCPConversationContext(
	value mcpSpec.MCPConversationContext,
) bool {
	return len(value.Servers) == 0 &&
		len(value.Resources) == 0 &&
		len(value.ResourceTemplates) == 0 &&
		len(value.Prompts) == 0
}

func mcpReferenceKey(
	server artifact.ArtifactRef,
	value string,
) string {
	return string(server.RootID) +
		"\x00" +
		string(server.ArtifactID) +
		"\x00" +
		value
}

package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func validateStartingMCPContextStructure(starting *mcpSpec.MCPConversationContext) error {
	if starting == nil {
		return nil
	}

	seenServers := make(map[string]struct{}, len(starting.Servers))
	for i, server := range starting.Servers {
		field := fmt.Sprintf("startingMCPContext.servers[%d]", i)
		if err := validateMCPServerSelectionStructure(field, server); err != nil {
			return err
		}

		key := mcpServerSelectionKey(server.BundleID, server.ServerID)
		if _, exists := seenServers[key]; exists {
			return fmt.Errorf("%s: duplicate server selection", field)
		}
		seenServers[key] = struct{}{}
	}

	if err := validateMCPResourcesStructure(starting.Resources); err != nil {
		return err
	}
	if err := validateMCPResourceTemplatesStructure(starting.ResourceTemplates); err != nil {
		return err
	}
	if err := validateMCPPromptsStructure(starting.Prompts); err != nil {
		return err
	}

	return nil
}

func validateMCPServerSelectionStructure(
	field string,
	server mcpSpec.MCPServerSelection,
) error {
	if server.BundleID == "" || server.ServerID == "" {
		return fmt.Errorf("%s: bundleID and serverID are required", field)
	}

	switch server.ToolExposure {
	case "":
		if len(server.SelectedTools) != 0 {
			return fmt.Errorf(
				"%s: toolExposure must be %q when selectedTools is non-empty",
				field,
				mcpSpec.MCPToolExposureSelected,
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
				"%s: selectedTools must be non-empty when toolExposure is %q",
				field,
				mcpSpec.MCPToolExposureSelected,
			)
		}
	default:
		return fmt.Errorf("%s: invalid toolExposure %q", field, server.ToolExposure)
	}

	seenTools := make(map[string]struct{}, len(server.SelectedTools))
	for i, tool := range server.SelectedTools {
		toolField := fmt.Sprintf("%s.selectedTools[%d]", field, i)
		if err := validateMCPToolSelectionStructure(toolField, server, tool); err != nil {
			return err
		}

		key := strings.TrimSpace(tool.ToolName)
		if _, exists := seenTools[key]; exists {
			return fmt.Errorf("%s: duplicate toolName %q", toolField, tool.ToolName)
		}
		seenTools[key] = struct{}{}
	}

	return nil
}

func validateMCPToolSelectionStructure(
	field string,
	parent mcpSpec.MCPServerSelection,
	selection mcpSpec.MCPToolSelection,
) error {
	if selection.BundleID == "" || selection.ServerID == "" || strings.TrimSpace(selection.ToolName) == "" {
		return fmt.Errorf("%s: bundleID, serverID and toolName are required", field)
	}
	if selection.BundleID != parent.BundleID || selection.ServerID != parent.ServerID {
		return fmt.Errorf("%s: selected tool must reference the parent server", field)
	}
	if err := validateOptionalMCPApprovalRule(field+".approvalRule", selection.ApprovalRule); err != nil {
		return err
	}
	if err := validateOptionalMCPExecutionMode(field+".executionMode", selection.ExecutionMode); err != nil {
		return err
	}
	return nil
}

func validateMCPResourcesStructure(resources []mcpSpec.MCPResourceRef) error {
	seen := make(map[string]struct{}, len(resources))
	for i, resource := range resources {
		field := fmt.Sprintf("startingMCPContext.resources[%d]", i)
		if resource.BundleID == "" || resource.ServerID == "" || strings.TrimSpace(resource.URI) == "" {
			return fmt.Errorf("%s: bundleID, serverID and uri are required", field)
		}

		key := strings.Join(
			[]string{string(resource.BundleID), string(resource.ServerID), resource.URI},
			"\x00",
		)
		if _, exists := seen[key]; exists {
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
	for i, selection := range templates {
		field := fmt.Sprintf("startingMCPContext.resourceTemplates[%d]", i)
		ref := selection.MCPResourceTemplateRef
		if ref.BundleID == "" || ref.ServerID == "" || strings.TrimSpace(ref.URITemplate) == "" {
			return fmt.Errorf("%s: bundleID, serverID and uriTemplate are required", field)
		}
		if err := validateMCPArgumentValuesStructure(field+".argumentValues", selection.ArgumentValues); err != nil {
			return err
		}
		if err := validateMCPRequiredArguments(
			field+".argumentValues",
			ref.Arguments,
			selection.ArgumentValues,
		); err != nil {
			return err
		}

		key := strings.Join(
			[]string{string(ref.BundleID), string(ref.ServerID), ref.URITemplate},
			"\x00",
		)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%s: duplicate resource template", field)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateMCPPromptsStructure(prompts []mcpSpec.MCPPromptSelection) error {
	seen := make(map[string]struct{}, len(prompts))
	for i, selection := range prompts {
		field := fmt.Sprintf("startingMCPContext.prompts[%d]", i)
		if selection.BundleID == "" || selection.ServerID == "" || strings.TrimSpace(selection.PromptName) == "" {
			return fmt.Errorf("%s: bundleID, serverID and promptName are required", field)
		}
		if err := validateMCPArgumentValuesStructure(field+".argumentValues", selection.ArgumentValues); err != nil {
			return err
		}
		if err := validateMCPRequiredArguments(
			field+".argumentValues",
			selection.Arguments,
			selection.ArgumentValues,
		); err != nil {
			return err
		}

		key := strings.Join(
			[]string{string(selection.BundleID), string(selection.ServerID), selection.PromptName},
			"\x00",
		)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%s: duplicate prompt", field)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateMCPArgumentValuesStructure(field string, values map[string]string) error {
	for name := range values {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%s: argument name is empty", field)
		}
	}
	return nil
}

func validateMCPRequiredArguments(
	field string,
	defs map[string]mcpSpec.MCPArgumentDefinition,
	values map[string]string,
) error {
	for name, def := range defs {
		if !def.Required {
			continue
		}

		argName := strings.TrimSpace(def.Name)
		if argName == "" {
			argName = strings.TrimSpace(name)
		}
		if argName == "" {
			continue
		}
		if strings.TrimSpace(values[argName]) == "" {
			return fmt.Errorf("%s: missing required argument %q", field, argName)
		}
	}
	return nil
}

func validateOptionalMCPApprovalRule(field string, rule *mcpSpec.MCPApprovalRule) error {
	if rule == nil {
		return nil
	}
	switch *rule {
	case mcpSpec.MCPApprovalRuleAsk, mcpSpec.MCPApprovalRuleAllow, mcpSpec.MCPApprovalRuleDeny:
		return nil
	default:
		return fmt.Errorf("%s: invalid approvalRule %q", field, *rule)
	}
}

func validateOptionalMCPExecutionMode(field string, mode *mcpSpec.MCPExecutionMode) error {
	if mode == nil {
		return nil
	}
	switch *mode {
	case mcpSpec.MCPExecutionModeManual, mcpSpec.MCPExecutionModeAuto:
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

func isEmptyMCPConversationContext(value mcpSpec.MCPConversationContext) bool {
	return len(value.Servers) == 0 &&
		len(value.Resources) == 0 &&
		len(value.ResourceTemplates) == 0 &&
		len(value.Prompts) == 0
}

func mcpServerSelectionKey(bundleID, serverID any) string {
	return fmt.Sprintf("%v\x00%v", bundleID, serverID)
}

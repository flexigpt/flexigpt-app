package runtime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

// ValidateMCPProviderToolMappingsForContext validates durable provider-tool
// mappings against the exact durable MCP context that authorized their
// inference exposure.
//
// This belongs in MCP validation, not in Conversation storage. Conversation
// owns persistence, while MCP owns the meaning of a mapping.
func ValidateMCPProviderToolMappingsForContext(
	contextValue MCPConversationContext,
	mappings []MCPProviderToolMapping,
) error {
	if err := ValidateMCPConversationContext(contextValue); err != nil {
		return err
	}

	servers := make(
		map[mcpSpec.ServerID]MCPServerSelection,
		len(contextValue.Servers),
	)
	for _, server := range contextValue.Servers {
		servers[server.Server] = server
	}

	providerNames := make(map[string]struct{}, len(mappings))
	choiceIDs := make(map[string]struct{}, len(mappings))
	for index, mapping := range mappings {
		if err := ValidateMCPProviderToolMapping(mapping); err != nil {
			return fmt.Errorf("MCP provider tool mappings[%d]: %w", index, err)
		}
		if _, duplicate := providerNames[mapping.ProviderToolName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP provider tool name %q",
				mcpSpec.ErrInvalid,
				mapping.ProviderToolName,
			)
		}
		if _, duplicate := choiceIDs[mapping.ChoiceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP choice ID %q",
				mcpSpec.ErrInvalid,
				mapping.ChoiceID,
			)
		}
		providerNames[mapping.ProviderToolName] = struct{}{}
		choiceIDs[mapping.ChoiceID] = struct{}{}

		server, selected := servers[mapping.Server]
		if !selected {
			return fmt.Errorf(
				"%w: mapped MCP tool %q belongs to an unselected server",
				mcpSpec.ErrInvalid,
				mapping.ToolName,
			)
		}

		switch server.ToolExposure {
		case MCPToolExposureSelected:
			selection, found := selectedTool(server.SelectedTools, mapping.ToolName)
			if !found {
				return fmt.Errorf(
					"%w: mapped MCP tool %q was not selected",
					mcpSpec.ErrInvalid,
					mapping.ToolName,
				)
			}
			if err := mappingMatchesSelection(mapping, selection); err != nil {
				return err
			}

		case MCPToolExposureAll:
			// SelectedTools is optional for "all". When a matching entry is
			// present, however, its conversation-level tightening still
			// applies to the durable provider mapping.
			if selection, found := selectedTool(server.SelectedTools, mapping.ToolName); found {
				if err := mappingMatchesSelection(mapping, selection); err != nil {
					return err
				}
			}

		default:
			return fmt.Errorf(
				"%w: mapped MCP tool %q has no enabled tool exposure",
				mcpSpec.ErrInvalid,
				mapping.ToolName,
			)
		}
	}
	return nil
}

// ValidateMCPAppContextUpdatesForContext binds App-originated model context
// updates to the exact durable MCP server selection that authorized them.
func ValidateMCPAppContextUpdatesForContext(
	contextValue MCPConversationContext,
	updates []MCPAppModelContextUpdate,
) error {
	if err := ValidateMCPConversationContext(contextValue); err != nil {
		return err
	}
	if err := ValidateMCPAppContextUpdates(updates); err != nil {
		return err
	}

	servers := make(
		map[mcpSpec.ServerID]struct{},
		len(contextValue.Servers),
	)
	for _, server := range contextValue.Servers {
		servers[server.Server] = struct{}{}
	}
	for index, update := range updates {
		if _, selected := servers[update.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP App context update %d belongs to an unselected server",
				mcpSpec.ErrInvalid,
				index,
			)
		}
	}
	return nil
}

func selectedTool(
	values []MCPToolSelection,
	name string,
) (MCPToolSelection, bool) {
	for _, value := range values {
		if value.ToolName == name {
			return value, true
		}
	}
	return MCPToolSelection{}, false
}

func mappingMatchesSelection(
	mapping MCPProviderToolMapping,
	selection MCPToolSelection,
) error {
	if selection.ProviderToolName != "" &&
		selection.ProviderToolName != mapping.ProviderToolName {
		return fmt.Errorf(
			"%w: mapped MCP provider tool identity changed for %q",
			mcpSpec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.ChoiceID != "" && selection.ChoiceID != mapping.ChoiceID {
		return fmt.Errorf(
			"%w: mapped MCP choice identity changed for %q",
			mcpSpec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.Digest != "" && selection.Digest != mapping.ToolDigest {
		return fmt.Errorf(
			"%w: mapped MCP tool digest changed for %q",
			mcpSpec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.AppResourceURI != "" &&
		selection.AppResourceURI != mapping.AppResourceURI {
		return fmt.Errorf(
			"%w: mapped MCP App resource changed for %q",
			mcpSpec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if len(selection.Visibility) != 0 &&
		!sameVisibility(selection.Visibility, mapping.Visibility) {
		return fmt.Errorf(
			"%w: mapped MCP App visibility changed for %q",
			mcpSpec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.ApprovalRule != nil &&
		mcpSpec.ApprovalRuleRank(mapping.ApprovalRule) < mcpSpec.ApprovalRuleRank(*selection.ApprovalRule) {
		return fmt.Errorf(
			"%w: mapped MCP approval rule weakens conversation policy",
			mcpSpec.ErrInvalid,
		)
	}
	if selection.ExecutionMode != nil &&
		mcpSpec.ExecutionModeRank(mapping.ExecutionMode) < mcpSpec.ExecutionModeRank(*selection.ExecutionMode) {
		return fmt.Errorf(
			"%w: mapped MCP execution mode weakens conversation policy",
			mcpSpec.ErrInvalid,
		)
	}
	return nil
}

func sameVisibility(left, right []string) bool {
	leftSet := normalizedVisibilitySet(left)
	rightSet := normalizedVisibilitySet(right)
	if len(leftSet) != len(rightSet) {
		return false
	}
	for value := range leftSet {
		if _, found := rightSet[value]; !found {
			return false
		}
	}
	return true
}

func normalizedVisibilitySet(values []string) map[string]struct{} {
	output := make(map[string]struct{}, len(values))
	for _, value := range values {
		output[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
	}
	return output
}

// ValidateMCPConversationContext validates durable MCP conversation selection
// structure without requiring a live MCP runtime connection.
func ValidateMCPConversationContext(value MCPConversationContext) error {
	if len(value.Servers) > mcpSpec.MaxDiscoveryCandidates ||
		len(value.Resources) > mcpSpec.MaxDiscoveryCandidates ||
		len(value.ResourceTemplates) > mcpSpec.MaxDiscoveryCandidates ||
		len(value.Prompts) > mcpSpec.MaxDiscoveryCandidates {
		return fmt.Errorf("%w: MCP conversation context exceeds entry limits", mcpSpec.ErrInvalid)
	}

	servers := make(map[mcpSpec.ServerID]struct{}, len(value.Servers))
	for index, selection := range value.Servers {
		if err := validateMCPServerSelection(selection); err != nil {
			return fmt.Errorf("MCP servers[%d]: %w", index, err)
		}
		if _, duplicate := servers[selection.Server]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP server selection %q",
				mcpSpec.ErrInvalid,
				selection.Server,
			)
		}
		servers[selection.Server] = struct{}{}
	}

	resources := make(map[string]struct{}, len(value.Resources))
	for index, resource := range value.Resources {
		if err := validateMCPResourceRef(resource); err != nil {
			return fmt.Errorf("MCP resources[%d]: %w", index, err)
		}
		if _, selected := servers[resource.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP resource %q belongs to a server not selected by the conversation",
				mcpSpec.ErrInvalid,
				resource.URI,
			)
		}
		key := mcpReferenceKey(resource.Server, resource.URI)
		if _, duplicate := resources[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource %q",
				mcpSpec.ErrInvalid,
				resource.URI,
			)
		}
		resources[key] = struct{}{}
	}

	templates := make(
		map[string]struct{},
		len(value.ResourceTemplates),
	)
	for index, selection := range value.ResourceTemplates {
		if err := validateMCPResourceTemplateSelection(selection); err != nil {
			return fmt.Errorf(
				"MCP resource templates[%d]: %w",
				index,
				err,
			)
		}
		if _, selected := servers[selection.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP resource template %q belongs to a server not selected by the conversation",
				mcpSpec.ErrInvalid,
				selection.URITemplate,
			)
		}
		key := mcpReferenceKey(
			selection.Server,
			selection.URITemplate,
		)
		if _, duplicate := templates[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource template %q",
				mcpSpec.ErrInvalid,
				selection.URITemplate,
			)
		}
		templates[key] = struct{}{}
	}

	prompts := make(map[string]struct{}, len(value.Prompts))
	for index, selection := range value.Prompts {
		if err := validateMCPPromptSelection(selection); err != nil {
			return fmt.Errorf("MCP prompts[%d]: %w", index, err)
		}
		if _, selected := servers[selection.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP prompt %q belongs to a server not selected by the conversation",
				mcpSpec.ErrInvalid,
				selection.PromptName,
			)
		}
		key := mcpReferenceKey(
			selection.Server,
			selection.PromptName,
		)
		if _, duplicate := prompts[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP prompt %q",
				mcpSpec.ErrInvalid,
				selection.PromptName,
			)
		}
		prompts[key] = struct{}{}
	}

	return nil
}

// ValidateMCPAppContextUpdates validates application-originated MCP App
// context updates before they are inserted into model input.
func ValidateMCPAppContextUpdates(
	values []MCPAppModelContextUpdate,
) error {
	for index, value := range values {
		if err := value.Server.Validate(); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
		if err := mcpSpec.ValidateOptionalText(
			"MCP App instance ID",
			value.InstanceID,
			mcpSpec.MaxDisplayNameBytes,
		); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
		if err := mcpSpec.ValidateOptionalText(
			"MCP App resource URI",
			value.ResourceURI,
			mcpSpec.MaxURIBytes,
		); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
	}
	return nil
}

// ValidateMCPProviderToolMapping validates one durable provider-tool mapping emitted during MCP
// inference hydration. These mappings bind later model tool calls to a
// specific Artifact-backed MCP server and discovered tool digest.
func ValidateMCPProviderToolMapping(m MCPProviderToolMapping) error {
	if err := m.Server.Validate(); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP provider tool name",
		m.ProviderToolName,
		mcpSpec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP provider choice ID",
		m.ChoiceID,
		mcpSpec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP tool name",
		m.ToolName,
		mcpSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP tool digest",
		m.ToolDigest,
		mcpSpec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateMCPApprovalRule(m.ApprovalRule); err != nil {
		return err
	}
	if err := mcpSpec.ValidateMCPExecutionMode(m.ExecutionMode); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP App resource URI",
		m.AppResourceURI,
		mcpSpec.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(m.Visibility)
}

func validateMCPServerSelection(value MCPServerSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP snapshot digest",
		value.SnapshotDigest,
		mcpSpec.MaxFingerprintBytes,
	); err != nil {
		return err
	}

	switch value.ToolExposure {
	case MCPToolExposureNone:
		if len(value.SelectedTools) != 0 {
			return fmt.Errorf(
				"%w: selected MCP tools require selected tool exposure",
				mcpSpec.ErrInvalid,
			)
		}

	case MCPToolExposureSelected:
		if len(value.SelectedTools) == 0 {
			return fmt.Errorf(
				"%w: selected MCP tool exposure requires tools",
				mcpSpec.ErrInvalid,
			)
		}

	case MCPToolExposureAll:
		// An all-tools selection does not require a redundant tool list.

	default:
		return fmt.Errorf(
			"%w: invalid MCP tool exposure %q",
			mcpSpec.ErrInvalid,
			value.ToolExposure,
		)
	}

	tools := make(map[string]struct{}, len(value.SelectedTools))
	for index, tool := range value.SelectedTools {
		if err := validateMCPToolSelection(tool); err != nil {
			return fmt.Errorf("selected tools[%d]: %w", index, err)
		}
		if tool.Server != value.Server {
			return fmt.Errorf(
				"%w: selected MCP tool belongs to another server",
				mcpSpec.ErrInvalid,
			)
		}
		if _, duplicate := tools[tool.ToolName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate selected MCP tool %q",
				mcpSpec.ErrInvalid,
				tool.ToolName,
			)
		}
		tools[tool.ToolName] = struct{}{}
	}
	return nil
}

func validateMCPToolSelection(value MCPToolSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP tool name",
		value.ToolName,
		mcpSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP provider tool name",
		value.ProviderToolName,
		mcpSpec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP tool choice ID",
		value.ChoiceID,
		mcpSpec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP tool digest",
		value.Digest,
		mcpSpec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if value.ApprovalRule != nil {
		if err := mcpSpec.ValidateMCPApprovalRule(*value.ApprovalRule); err != nil {
			return err
		}
	}
	if value.ExecutionMode != nil {
		if err := mcpSpec.ValidateMCPExecutionMode(*value.ExecutionMode); err != nil {
			return err
		}
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP App resource URI",
		value.AppResourceURI,
		mcpSpec.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(value.Visibility)
}

func validateMCPResourceRef(value MCPResourceRef) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP resource URI",
		value.URI,
		mcpSpec.MaxURIBytes,
	); err != nil {
		return err
	}
	return mcpSpec.ValidateOptionalText(
		"MCP resource digest",
		value.Digest,
		mcpSpec.MaxFingerprintBytes,
	)
}

func validateMCPResourceTemplateSelection(
	value MCPResourceTemplateSelection,
) error {
	ref := value.MCPResourceTemplateRef
	if err := ref.Server.Validate(); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP resource URI template",
		ref.URITemplate,
		mcpSpec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP resource template digest",
		ref.Digest,
		mcpSpec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := validateMCPArgumentValues(value.ArgumentValues); err != nil {
		return err
	}
	return validateMCPArgumentDefinitions(ref.Arguments)
}

func validateMCPPromptSelection(value MCPPromptSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP prompt name",
		value.PromptName,
		mcpSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP prompt digest",
		value.Digest,
		mcpSpec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := validateMCPArgumentValues(value.ArgumentValues); err != nil {
		return err
	}
	return validateMCPArgumentDefinitions(value.Arguments)
}

func validateMCPArgumentDefinitions(
	values map[string]MCPArgumentDefinition,
) error {
	for name, value := range values {
		if err := mcpSpec.ValidateRequiredText(
			"MCP argument name",
			name,
			mcpSpec.MaxKindBytes,
		); err != nil {
			return err
		}
		if value.Name != "" && value.Name != name {
			return fmt.Errorf(
				"%w: MCP argument definition key %q differs from name %q",
				mcpSpec.ErrInvalid,
				name,
				value.Name,
			)
		}
	}
	return nil
}

func validateMCPArgumentValues(values map[string]string) error {
	for name, value := range values {
		if err := mcpSpec.ValidateRequiredText(
			"MCP argument value name",
			name,
			mcpSpec.MaxKindBytes,
		); err != nil {
			return err
		}
		if !utf8.ValidString(value) ||
			len(value) > mcpSpec.MaxDescriptionBytes {
			return fmt.Errorf(
				"%w: MCP argument value %q is invalid",
				mcpSpec.ErrInvalid,
				name,
			)
		}
	}
	return nil
}

func validateMCPVisibility(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		switch value {
		case "model", "app":
		default:
			return fmt.Errorf(
				"%w: invalid MCP App visibility %q",
				mcpSpec.ErrInvalid,
				raw,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP App visibility %q",
				mcpSpec.ErrInvalid,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func mcpReferenceKey(
	server mcpSpec.ServerID,
	value string,
) string {
	return string(server) + "\x00" + value
}

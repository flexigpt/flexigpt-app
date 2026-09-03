package conversation

import (
	"fmt"
	"strings"
	"unicode/utf8"

	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
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
		map[mcpServer.ServerID]MCPServerSelection,
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
				mcpServer.ErrInvalid,
				mapping.ProviderToolName,
			)
		}
		if _, duplicate := choiceIDs[mapping.ChoiceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP choice ID %q",
				mcpServer.ErrInvalid,
				mapping.ChoiceID,
			)
		}
		providerNames[mapping.ProviderToolName] = struct{}{}
		choiceIDs[mapping.ChoiceID] = struct{}{}

		server, selected := servers[mapping.Server]
		if !selected {
			return fmt.Errorf(
				"%w: mapped MCP tool %q belongs to an unselected server",
				mcpServer.ErrInvalid,
				mapping.ToolName,
			)
		}

		switch server.ToolExposure {
		case MCPToolExposureSelected:
			selection, found := selectedTool(server.SelectedTools, mapping.ToolName)
			if !found {
				return fmt.Errorf(
					"%w: mapped MCP tool %q was not selected",
					mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
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
		map[mcpServer.ServerID]struct{},
		len(contextValue.Servers),
	)
	for _, server := range contextValue.Servers {
		servers[server.Server] = struct{}{}
	}
	for index, update := range updates {
		if _, selected := servers[update.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP App context update %d belongs to an unselected server",
				mcpServer.ErrInvalid,
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
			mcpServer.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.ChoiceID != "" && selection.ChoiceID != mapping.ChoiceID {
		return fmt.Errorf(
			"%w: mapped MCP choice identity changed for %q",
			mcpServer.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.Digest != "" && selection.Digest != mapping.ToolDigest {
		return fmt.Errorf(
			"%w: mapped MCP tool digest changed for %q",
			mcpServer.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.AppResourceURI != "" &&
		selection.AppResourceURI != mapping.AppResourceURI {
		return fmt.Errorf(
			"%w: mapped MCP App resource changed for %q",
			mcpServer.ErrInvalid,
			mapping.ToolName,
		)
	}
	if len(selection.Visibility) != 0 &&
		!sameVisibility(selection.Visibility, mapping.Visibility) {
		return fmt.Errorf(
			"%w: mapped MCP App visibility changed for %q",
			mcpServer.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.ApprovalRule != nil &&
		mcpPolicy.ApprovalRuleRank(mapping.ApprovalRule) < mcpPolicy.ApprovalRuleRank(*selection.ApprovalRule) {
		return fmt.Errorf(
			"%w: mapped MCP approval rule weakens conversation policy",
			mcpServer.ErrInvalid,
		)
	}
	if selection.ExecutionMode != nil &&
		mcpPolicy.ExecutionModeRank(mapping.ExecutionMode) < mcpPolicy.ExecutionModeRank(*selection.ExecutionMode) {
		return fmt.Errorf(
			"%w: mapped MCP execution mode weakens conversation policy",
			mcpServer.ErrInvalid,
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
	if len(value.Servers) > mcpServer.MaxDiscoveryCandidates ||
		len(value.Resources) > mcpServer.MaxDiscoveryCandidates ||
		len(value.ResourceTemplates) > mcpServer.MaxDiscoveryCandidates ||
		len(value.Prompts) > mcpServer.MaxDiscoveryCandidates {
		return fmt.Errorf("%w: MCP conversation context exceeds entry limits", mcpServer.ErrInvalid)
	}

	servers := make(map[mcpServer.ServerID]struct{}, len(value.Servers))
	for index, selection := range value.Servers {
		if err := validateMCPServerSelection(selection); err != nil {
			return fmt.Errorf("MCP servers[%d]: %w", index, err)
		}
		if _, duplicate := servers[selection.Server]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP server selection %q",
				mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
				resource.URI,
			)
		}
		key := mcpReferenceKey(resource.Server, resource.URI)
		if _, duplicate := resources[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource %q",
				mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
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
		if err := mcpServer.ValidateOptionalText(
			"MCP App instance ID",
			value.InstanceID,
			mcpServer.MaxDisplayNameBytes,
		); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
		if err := mcpServer.ValidateOptionalText(
			"MCP App resource URI",
			value.ResourceURI,
			mcpServer.MaxURIBytes,
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
	if err := mcpServer.ValidateRequiredText(
		"MCP provider tool name",
		m.ProviderToolName,
		mcpServer.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateRequiredText(
		"MCP provider choice ID",
		m.ChoiceID,
		mcpServer.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateRequiredText(
		"MCP tool name",
		m.ToolName,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateRequiredText(
		"MCP tool digest",
		m.ToolDigest,
		mcpServer.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := mcpPolicy.ValidateMCPApprovalRule(m.ApprovalRule); err != nil {
		return err
	}
	if err := mcpPolicy.ValidateMCPExecutionMode(m.ExecutionMode); err != nil {
		return err
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP App resource URI",
		m.AppResourceURI,
		mcpServer.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(m.Visibility)
}

func validateMCPServerSelection(value MCPServerSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP snapshot digest",
		value.SnapshotDigest,
		mcpServer.MaxFingerprintBytes,
	); err != nil {
		return err
	}

	switch value.ToolExposure {
	case MCPToolExposureNone:
		if len(value.SelectedTools) != 0 {
			return fmt.Errorf(
				"%w: selected MCP tools require selected tool exposure",
				mcpServer.ErrInvalid,
			)
		}

	case MCPToolExposureSelected:
		if len(value.SelectedTools) == 0 {
			return fmt.Errorf(
				"%w: selected MCP tool exposure requires tools",
				mcpServer.ErrInvalid,
			)
		}

	case MCPToolExposureAll:
		// An all-tools selection does not require a redundant tool list.

	default:
		return fmt.Errorf(
			"%w: invalid MCP tool exposure %q",
			mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
			)
		}
		if _, duplicate := tools[tool.ToolName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate selected MCP tool %q",
				mcpServer.ErrInvalid,
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
	if err := mcpServer.ValidateRequiredText(
		"MCP tool name",
		value.ToolName,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP provider tool name",
		value.ProviderToolName,
		mcpServer.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP tool choice ID",
		value.ChoiceID,
		mcpServer.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP tool digest",
		value.Digest,
		mcpServer.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if value.ApprovalRule != nil {
		if err := mcpPolicy.ValidateMCPApprovalRule(*value.ApprovalRule); err != nil {
			return err
		}
	}
	if value.ExecutionMode != nil {
		if err := mcpPolicy.ValidateMCPExecutionMode(*value.ExecutionMode); err != nil {
			return err
		}
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP App resource URI",
		value.AppResourceURI,
		mcpServer.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(value.Visibility)
}

func validateMCPResourceRef(value mcpServer.MCPResourceRef) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := mcpServer.ValidateRequiredText(
		"MCP resource URI",
		value.URI,
		mcpServer.MaxURIBytes,
	); err != nil {
		return err
	}
	return mcpServer.ValidateOptionalText(
		"MCP resource digest",
		value.Digest,
		mcpServer.MaxFingerprintBytes,
	)
}

func validateMCPResourceTemplateSelection(
	value MCPResourceTemplateSelection,
) error {
	ref := value.MCPResourceTemplateRef
	if err := ref.Server.Validate(); err != nil {
		return err
	}
	if err := mcpServer.ValidateRequiredText(
		"MCP resource URI template",
		ref.URITemplate,
		mcpServer.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP resource template digest",
		ref.Digest,
		mcpServer.MaxFingerprintBytes,
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
	if err := mcpServer.ValidateRequiredText(
		"MCP prompt name",
		value.PromptName,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.ValidateOptionalText(
		"MCP prompt digest",
		value.Digest,
		mcpServer.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := validateMCPArgumentValues(value.ArgumentValues); err != nil {
		return err
	}
	return validateMCPArgumentDefinitions(value.Arguments)
}

func validateMCPArgumentDefinitions(
	values map[string]mcpServer.MCPArgumentDefinition,
) error {
	for name, value := range values {
		if err := mcpServer.ValidateRequiredText(
			"MCP argument name",
			name,
			mcpServer.MaxKindBytes,
		); err != nil {
			return err
		}
		if value.Name != "" && value.Name != name {
			return fmt.Errorf(
				"%w: MCP argument definition key %q differs from name %q",
				mcpServer.ErrInvalid,
				name,
				value.Name,
			)
		}
	}
	return nil
}

func validateMCPArgumentValues(values map[string]string) error {
	for name, value := range values {
		if err := mcpServer.ValidateRequiredText(
			"MCP argument value name",
			name,
			mcpServer.MaxKindBytes,
		); err != nil {
			return err
		}
		if !utf8.ValidString(value) ||
			len(value) > mcpServer.MaxDescriptionBytes {
			return fmt.Errorf(
				"%w: MCP argument value %q is invalid",
				mcpServer.ErrInvalid,
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
				mcpServer.ErrInvalid,
				raw,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP App visibility %q",
				mcpServer.ErrInvalid,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func mcpReferenceKey(
	server mcpServer.ServerID,
	value string,
) string {
	return string(server) + "\x00" + value
}

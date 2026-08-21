package runtime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
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
		map[artifact.ArtifactRef]MCPServerSelection,
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
				basespec.ErrInvalid,
				mapping.ProviderToolName,
			)
		}
		if _, duplicate := choiceIDs[mapping.ChoiceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP choice ID %q",
				basespec.ErrInvalid,
				mapping.ChoiceID,
			)
		}
		providerNames[mapping.ProviderToolName] = struct{}{}
		choiceIDs[mapping.ChoiceID] = struct{}{}

		server, selected := servers[mapping.Server]
		if !selected {
			return fmt.Errorf(
				"%w: mapped MCP tool %q belongs to an unselected server",
				basespec.ErrInvalid,
				mapping.ToolName,
			)
		}

		switch server.ToolExposure {
		case MCPToolExposureSelected:
			selection, found := selectedTool(server.SelectedTools, mapping.ToolName)
			if !found {
				return fmt.Errorf(
					"%w: mapped MCP tool %q was not selected",
					basespec.ErrInvalid,
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
				basespec.ErrInvalid,
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
		map[artifact.ArtifactRef]struct{},
		len(contextValue.Servers),
	)
	for _, server := range contextValue.Servers {
		servers[server.Server] = struct{}{}
	}
	for index, update := range updates {
		if _, selected := servers[update.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP App context update %d belongs to an unselected server",
				basespec.ErrInvalid,
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
			basespec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.ChoiceID != "" && selection.ChoiceID != mapping.ChoiceID {
		return fmt.Errorf(
			"%w: mapped MCP choice identity changed for %q",
			basespec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.Digest != "" && selection.Digest != mapping.ToolDigest {
		return fmt.Errorf(
			"%w: mapped MCP tool digest changed for %q",
			basespec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.AppResourceURI != "" &&
		selection.AppResourceURI != mapping.AppResourceURI {
		return fmt.Errorf(
			"%w: mapped MCP App resource changed for %q",
			basespec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if len(selection.Visibility) != 0 &&
		!sameVisibility(selection.Visibility, mapping.Visibility) {
		return fmt.Errorf(
			"%w: mapped MCP App visibility changed for %q",
			basespec.ErrInvalid,
			mapping.ToolName,
		)
	}
	if selection.ApprovalRule != nil &&
		policy.ApprovalRuleRank(mapping.ApprovalRule) < policy.ApprovalRuleRank(*selection.ApprovalRule) {
		return fmt.Errorf(
			"%w: mapped MCP approval rule weakens conversation policy",
			basespec.ErrInvalid,
		)
	}
	if selection.ExecutionMode != nil &&
		policy.ExecutionModeRank(mapping.ExecutionMode) < policy.ExecutionModeRank(*selection.ExecutionMode) {
		return fmt.Errorf(
			"%w: mapped MCP execution mode weakens conversation policy",
			basespec.ErrInvalid,
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
	if len(value.Servers) > basespec.MaxDiscoveryCandidates ||
		len(value.Resources) > basespec.MaxDiscoveryCandidates ||
		len(value.ResourceTemplates) > basespec.MaxDiscoveryCandidates ||
		len(value.Prompts) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf("%w: MCP conversation context exceeds entry limits", basespec.ErrInvalid)
	}

	servers := make(map[artifact.ArtifactRef]struct{}, len(value.Servers))
	for index, selection := range value.Servers {
		if err := validateMCPServerSelection(selection); err != nil {
			return fmt.Errorf("MCP servers[%d]: %w", index, err)
		}
		if _, duplicate := servers[selection.Server]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP server selection %q",
				basespec.ErrInvalid,
				selection.Server.ArtifactID,
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
				basespec.ErrInvalid,
				resource.URI,
			)
		}
		key := mcpReferenceKey(resource.Server, resource.URI)
		if _, duplicate := resources[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource %q",
				basespec.ErrInvalid,
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
				basespec.ErrInvalid,
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
				basespec.ErrInvalid,
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
				basespec.ErrInvalid,
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
				basespec.ErrInvalid,
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
		if err := basespec.ValidateOptionalText(
			"MCP App instance ID",
			value.InstanceID,
			basespec.MaxDisplayNameBytes,
		); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
		if err := basespec.ValidateOptionalText(
			"MCP App resource URI",
			value.ResourceURI,
			basespec.MaxURIBytes,
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
	if err := basespec.ValidateRequiredText(
		"MCP provider tool name",
		m.ProviderToolName,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP provider choice ID",
		m.ChoiceID,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP tool name",
		m.ToolName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP tool digest",
		m.ToolDigest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := policy.ValidateMCPApprovalRule(m.ApprovalRule); err != nil {
		return err
	}
	if err := policy.ValidateMCPExecutionMode(m.ExecutionMode); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP App resource URI",
		m.AppResourceURI,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(m.Visibility)
}

func validateMCPServerSelection(value MCPServerSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP snapshot digest",
		value.SnapshotDigest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}

	switch value.ToolExposure {
	case MCPToolExposureNone:
		if len(value.SelectedTools) != 0 {
			return fmt.Errorf(
				"%w: selected MCP tools require selected tool exposure",
				basespec.ErrInvalid,
			)
		}

	case MCPToolExposureSelected:
		if len(value.SelectedTools) == 0 {
			return fmt.Errorf(
				"%w: selected MCP tool exposure requires tools",
				basespec.ErrInvalid,
			)
		}

	case MCPToolExposureAll:
		// An all-tools selection does not require a redundant tool list.

	default:
		return fmt.Errorf(
			"%w: invalid MCP tool exposure %q",
			basespec.ErrInvalid,
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
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := tools[tool.ToolName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate selected MCP tool %q",
				basespec.ErrInvalid,
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
	if err := basespec.ValidateRequiredText(
		"MCP tool name",
		value.ToolName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP provider tool name",
		value.ProviderToolName,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP tool choice ID",
		value.ChoiceID,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP tool digest",
		value.Digest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if value.ApprovalRule != nil {
		if err := policy.ValidateMCPApprovalRule(*value.ApprovalRule); err != nil {
			return err
		}
	}
	if value.ExecutionMode != nil {
		if err := policy.ValidateMCPExecutionMode(*value.ExecutionMode); err != nil {
			return err
		}
	}
	if err := basespec.ValidateOptionalText(
		"MCP App resource URI",
		value.AppResourceURI,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(value.Visibility)
}

func validateMCPResourceRef(value MCPResourceRef) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP resource URI",
		value.URI,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return basespec.ValidateOptionalText(
		"MCP resource digest",
		value.Digest,
		basespec.MaxFingerprintBytes,
	)
}

func validateMCPResourceTemplateSelection(
	value MCPResourceTemplateSelection,
) error {
	ref := value.MCPResourceTemplateRef
	if err := ref.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP resource URI template",
		ref.URITemplate,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP resource template digest",
		ref.Digest,
		basespec.MaxFingerprintBytes,
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
	if err := basespec.ValidateRequiredText(
		"MCP prompt name",
		value.PromptName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP prompt digest",
		value.Digest,
		basespec.MaxFingerprintBytes,
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
		if err := basespec.ValidateRequiredText(
			"MCP argument name",
			name,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if value.Name != "" && value.Name != name {
			return fmt.Errorf(
				"%w: MCP argument definition key %q differs from name %q",
				basespec.ErrInvalid,
				name,
				value.Name,
			)
		}
	}
	return nil
}

func validateMCPArgumentValues(values map[string]string) error {
	for name, value := range values {
		if err := basespec.ValidateRequiredText(
			"MCP argument value name",
			name,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if !utf8.ValidString(value) ||
			len(value) > basespec.MaxDescriptionBytes {
			return fmt.Errorf(
				"%w: MCP argument value %q is invalid",
				basespec.ErrInvalid,
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
				basespec.ErrInvalid,
				raw,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP App visibility %q",
				basespec.ErrInvalid,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
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

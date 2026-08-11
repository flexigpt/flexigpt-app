package validate

import (
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

// ValidateMCPProviderToolMappingsForContext validates durable provider-tool
// mappings against the exact durable MCP context that authorized their
// inference exposure.
//
// This belongs in MCP validation, not in Conversation storage. Conversation
// owns persistence, while MCP owns the meaning of a mapping.
func ValidateMCPProviderToolMappingsForContext(
	contextValue spec.MCPConversationContext,
	mappings []spec.MCPProviderToolMapping,
) error {
	if err := ValidateMCPConversationContext(contextValue); err != nil {
		return err
	}

	servers := make(
		map[artifact.ArtifactRef]spec.MCPServerSelection,
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
		case spec.MCPToolExposureAll:
			continue

		case spec.MCPToolExposureSelected:
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
	contextValue spec.MCPConversationContext,
	updates []spec.MCPAppModelContextUpdate,
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
	values []spec.MCPToolSelection,
	name string,
) (spec.MCPToolSelection, bool) {
	for _, value := range values {
		if value.ToolName == name {
			return value, true
		}
	}
	return spec.MCPToolSelection{}, false
}

func mappingMatchesSelection(
	mapping spec.MCPProviderToolMapping,
	selection spec.MCPToolSelection,
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
		approvalRank(mapping.ApprovalRule) < approvalRank(*selection.ApprovalRule) {
		return fmt.Errorf(
			"%w: mapped MCP approval rule weakens conversation policy",
			basespec.ErrInvalid,
		)
	}
	if selection.ExecutionMode != nil &&
		executionRank(mapping.ExecutionMode) < executionRank(*selection.ExecutionMode) {
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

func approvalRank(value spec.MCPApprovalRule) int {
	switch value {
	case spec.MCPApprovalRuleDeny:
		return 3
	case spec.MCPApprovalRuleAsk:
		return 2
	default:
		return 1
	}
}

func executionRank(value spec.MCPExecutionMode) int {
	if value == spec.MCPExecutionModeManual {
		return 2
	}
	return 1
}

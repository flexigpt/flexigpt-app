package store

import (
	"context"
	"errors"
	"fmt"

	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
)

func validateStartingMCPContextStructure(
	starting *mcpRuntime.MCPConversationContext,
) error {
	if starting == nil {
		return nil
	}
	if err := mcpRuntime.ValidateMCPConversationContext(*starting); err != nil {
		return fmt.Errorf("startingMCPContext: %w", err)
	}
	return nil
}

func validateStartingMCPContextReferences(
	ctx context.Context,
	starting *mcpRuntime.MCPConversationContext,
	lookups ReferenceLookups,
) error {
	if starting == nil || isEmptyMCPConversationContext(*starting) {
		return nil
	}
	if lookups.MCPContext == nil {
		return errors.New("mcp context lookup not configured")
	}
	if err := lookups.MCPContext.ValidateMCPConversationContext(
		ctx,
		*starting,
	); err != nil {
		return fmt.Errorf("startingMCPContext: %w", err)
	}
	return nil
}

func isEmptyMCPConversationContext(
	value mcpRuntime.MCPConversationContext,
) bool {
	return len(value.Servers) == 0 &&
		len(value.Resources) == 0 &&
		len(value.ResourceTemplates) == 0 &&
		len(value.Prompts) == 0
}

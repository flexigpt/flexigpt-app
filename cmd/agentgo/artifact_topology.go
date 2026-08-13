package main

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
)

const (
	defaultWorkspaceRootID basespec.RootID = "0198f097-0d5b-7000-8000-000000000001"
	mcpUserRootID          basespec.RootID = "0198f097-0d5b-7000-8000-000000000002"
	mcpBuiltInRootID       basespec.RootID = "0198f097-0d5b-7000-8000-000000000003"

	defaultWorkspaceRootStorageKey basespec.StorageKey = "workspaces"
	mcpUserRootStorageKey          basespec.StorageKey = "mcp"

	defaultWorkspaceRootDisplayName = "FlexiGPT Workspaces"
	defaultWorkspaceRootDescription = "Local namespace for user Workspace collections."

	mcpUserRootDisplayName = "FlexiGPT MCP Bundles"
	mcpUserRootDescription = "Local namespace for user-managed MCP Bundles."
)

func defaultWorkspaceRootDraft() root.RootDraft {
	return root.RootDraft{
		ID:          defaultWorkspaceRootID,
		StorageKey:  defaultWorkspaceRootStorageKey,
		DisplayName: defaultWorkspaceRootDisplayName,
		Description: defaultWorkspaceRootDescription,
	}
}

func defaultMCPUserRootDraft() root.RootDraft {
	return root.RootDraft{
		ID:          mcpUserRootID,
		StorageKey:  mcpUserRootStorageKey,
		DisplayName: mcpUserRootDisplayName,
		Description: mcpUserRootDescription,
	}
}

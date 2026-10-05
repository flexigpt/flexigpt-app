package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
)

func PluginProfile() plugin.Profile {
	return plugin.Profile{
		Name:                "mcp",
		SourceStorageKey:    plugin.MCPManagedPluginSourceStorageKey,
		SourceDisplayName:   "User-managed MCP artifacts",
		BaselineName:        plugin.MCPBaselinePluginName,
		BaselineDisplayName: "MCP Baseline",
		BaselineDescription: "Application-provisioned editable MCP Plugin.",
		PackageKind:         plugin.ManagedCollectionPackageKind,
		MembershipPolicy: pluginDomain.MembershipPolicy{
			Mode: pluginDomain.MembershipModeMixedType,
			AllowedTypes: []declaration.Type{
				declaration.Type("mcp"),
				declaration.Type("mcp.policy"),
			},
			AllowedForms: []declaration.MemberForm{
				declaration.MemberNamed,
			},
		},
	}
}

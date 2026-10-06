package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
)

func PluginProfile() pluginAPI.Profile {
	return pluginAPI.Profile{
		Name:                "mcp",
		SourceStorageKey:    pluginAPI.MCPManagedPluginSourceStorageKey,
		SourceDisplayName:   "User-managed MCP artifacts",
		BaselineName:        pluginAPI.MCPBaselinePluginName,
		BaselineDisplayName: "MCP Baseline",
		BaselineDescription: "Application-provisioned editable MCP Plugin.",
		PackageKind:         pluginAPI.ManagedPluginPackageKind,
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

package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

func PluginProfile() plugin.Profile {
	return plugin.Profile{
		Name:        "tool",
		ReadOnly:    true,
		PackageKind: ToolCollectionPackageKind,
		DocumentUse: topology.DocumentUseToolPlugin,
		MembershipPolicy: pluginDomain.MembershipPolicy{
			Mode: pluginDomain.MembershipModeSingleType,
			AllowedTypes: []declaration.Type{
				toolv1.ToolType,
			},
			AllowedForms: []declaration.MemberForm{
				declaration.MemberNamed,
			},
		},
	}
}

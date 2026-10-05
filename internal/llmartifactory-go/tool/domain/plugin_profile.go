package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
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
		ValidateDocument: func(document pluginv1.PluginDocument) error {
			_, err := ValidateToolCollectionDocument(document)
			return err
		},
	}
}

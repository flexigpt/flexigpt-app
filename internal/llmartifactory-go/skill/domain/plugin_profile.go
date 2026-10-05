package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
)

func PluginProfile() plugin.Profile {
	return plugin.Profile{
		Name:                "skill",
		SourceStorageKey:    plugin.SkillManagedPluginSourceStorageKey,
		SourceDisplayName:   "User-managed Skills",
		BaselineName:        plugin.SkillBaselinePluginName,
		BaselineDisplayName: "Skill Baseline",
		BaselineDescription: "Application-provisioned editable Skill Plugin.",
		PackageKind:         plugin.ManagedPluginPackageKind,
		MembershipPolicy: pluginDomain.MembershipPolicy{
			Mode: pluginDomain.MembershipModeSingleType,
			AllowedTypes: []declaration.Type{
				skillv1.SkillType,
			},
			AllowedForms: []declaration.MemberForm{
				declaration.MemberNamed,
			},
		},
	}
}

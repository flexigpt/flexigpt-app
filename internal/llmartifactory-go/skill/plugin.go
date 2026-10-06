package skill

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
)

func skillPluginProfile() pluginAPI.Profile {
	return pluginAPI.Profile{
		Name:                "skill",
		SourceStorageKey:    pluginAPI.SkillManagedPluginSourceStorageKey,
		SourceDisplayName:   "User-managed Skills",
		BaselineName:        pluginAPI.SkillBaselinePluginName,
		BaselineDisplayName: "Skill Baseline",
		BaselineDescription: "Application-provisioned editable Skill Plugin.",
		PackageKind:         pluginAPI.ManagedPluginPackageKind,
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

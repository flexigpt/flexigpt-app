package domain

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/domain"
)

const (
	AgentArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		agentv1.AgentType,
	)

	ManagedAgentPackageKind           managedpackageModel.PackageKind = "agent"
	BuiltinAgentCollectionPackageKind managedpackageModel.PackageKind = "agent-collection"
	AgentManagedSourceStorageKey      spec.StorageKey                 = "user-agents"
	AgentManagedCollectionPackageKind managedpackageModel.PackageKind = "plugin"
	AgentBaselineCollectionName       spec.LogicalName                = "agent-baseline"
	AgentSchemaID                     schemaModel.SchemaID            = agentv1.AgentSchemaID

	AgentSchemaVersion            = agentv1.AgentSchemaVersion
	AgentManagedSourceDisplayName = "User-managed Agents"
	AgentBaselineDisplayName      = "Agent Baseline"
	AgentBaselineDescription      = "Application-provisioned editable Agent Plugin."
	BuiltInInstallerName          = "agent.agent"
	HydrationSchemaVersion        = "agent.agent.builtin-hydration/v1"
)

func IsAgentKind(value artifactModel.ArtifactKind) bool {
	return value == AgentArtifactKind
}

func IsAgentSchema(value schemaModel.Key) bool {
	return value.Entity == schemaModel.EntityArtifact &&
		value.Kind == schemaModel.Kind(AgentArtifactKind) &&
		value.SchemaID == AgentSchemaID &&
		value.SchemaVersion == AgentSchemaVersion
}

func AgentPluginProfile() plugin.Profile {
	return plugin.Profile{
		Name:                "agent",
		SourceStorageKey:    AgentManagedSourceStorageKey,
		SourceDisplayName:   AgentManagedSourceDisplayName,
		BaselineName:        AgentBaselineCollectionName,
		BaselineDisplayName: AgentBaselineDisplayName,
		BaselineDescription: AgentBaselineDescription,
		PackageKind:         AgentManagedCollectionPackageKind,
		DocumentUse:         topology.DocumentUseAgentManagedPlugin,
		MembershipPolicy: pluginDomain.MembershipPolicy{
			Mode: pluginDomain.MembershipModeSingleType,
			AllowedTypes: []declaration.Type{
				agentv1.AgentType,
			},
			AllowedForms: []declaration.MemberForm{
				declaration.MemberNamed,
			},
		},
	}
}

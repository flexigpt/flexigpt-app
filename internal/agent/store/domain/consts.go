package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

const (
	AgentArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		agentv1.AgentType,
	)

	ManagedAgentPackageKind           sourceModel.PackageKind = "agent"
	BuiltinAgentCollectionPackageKind sourceModel.PackageKind = "agent-collection"
	AgentManagedSourceStorageKey      spec.StorageKey         = "user-agents"
	AgentManagedCollectionPackageKind sourceModel.PackageKind = "plugin"
	AgentBaselineCollectionName       spec.LogicalName        = "agent-baseline"
	AgentSchemaID                     schemaModel.SchemaID    = agentv1.AgentSchemaID

	AgentSchemaVersion            = agentv1.AgentSchemaVersion
	AgentManagedSourceDisplayName = "User-managed Agents"
	AgentBaselineDisplayName      = "Agent Baseline"
	AgentBaselineDescription      = "Application-provisioned editable Agent Collection."
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

func AgentCollectionDomainPolicy() collection.DomainPolicy {
	return collection.DomainPolicy{
		Name:                "agent",
		SourceStorageKey:    AgentManagedSourceStorageKey,
		SourceDisplayName:   AgentManagedSourceDisplayName,
		BaselineName:        AgentBaselineCollectionName,
		BaselineDisplayName: AgentBaselineDisplayName,
		BaselineDescription: AgentBaselineDescription,
		PackageKind:         AgentManagedCollectionPackageKind,
		DocumentUse:         documentTopology.DocumentUseAgentManagedCollection,
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeAgent,
		},
		AllowedMemberForms: []declaration.MemberForm{
			declaration.MemberNamed,
		},
	}
}

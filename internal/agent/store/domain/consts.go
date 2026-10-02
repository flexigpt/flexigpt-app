package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

const (
	AgentArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		agentv1.AgentType,
	)

	ManagedAgentPackageKind           source.PackageKind = "agent"
	BuiltinAgentCollectionPackageKind source.PackageKind = "agent-collection"
	AgentManagedSourceStorageKey      model.StorageKey   = "user-agents"
	AgentManagedCollectionPackageKind source.PackageKind = "plugin"
	AgentBaselineCollectionName       model.LogicalName  = "agent-baseline"
	AgentSchemaID                     schema.SchemaID    = agentv1.AgentSchemaID

	AgentSchemaVersion            = agentv1.AgentSchemaVersion
	AgentManagedSourceDisplayName = "User-managed Agents"
	AgentBaselineDisplayName      = "Agent Baseline"
	AgentBaselineDescription      = "Application-provisioned editable Agent Collection."
	BuiltInInstallerName          = "agent.agent"
	HydrationSchemaVersion        = "agent.agent.builtin-hydration/v1"
)

func IsAgentKind(value artifact.ArtifactKind) bool {
	return value == AgentArtifactKind
}

func IsAgentSchema(value schema.Key) bool {
	return value.Entity == schema.EntityArtifact &&
		value.Kind == schema.Kind(AgentArtifactKind) &&
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

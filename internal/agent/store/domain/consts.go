package domain

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

const (
	AgentArtifactKind artifact.ArtifactKind = artifact.ArtifactKind(
		agentv1.AgentType,
	)
	AgentSchemaID      schema.SchemaID = agentv1.AgentSchemaID
	AgentSchemaVersion                 = agentv1.AgentSchemaVersion

	AgentManagedSourceStorageKey basespec.StorageKey = "user-agents"

	AgentManagedSourceDisplayName = "User-managed Agents"

	AgentManagedCollectionPackageKind  source.PackageKind = "plugin"
	AgentManagedCollectionDocumentFile basespec.Locator   = "plugin.yaml"

	AgentBaselineCollectionName basespec.LogicalName = "agent-baseline"

	AgentBaselineDisplayName = "Agent Baseline"

	AgentBaselineDescription = "Application-provisioned editable Agent Collection."

	ManagedAgentPackageKind  source.PackageKind = "agent"
	ManagedAgentDocumentFile basespec.Locator   = "agent.yaml"

	BuiltinAgentCollectionPackageKind source.PackageKind = "agent-collection"
	BuiltinAgentDocumentFile          basespec.Locator   = "agent.yaml"

	BuiltInInstallerName   = "agent.agent"
	HydrationSchemaVersion = "agent.agent.builtin-hydration/v1"
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
		DocumentFile:        AgentManagedCollectionDocumentFile,
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeAgent,
		},
		AllowedMemberForms: []declaration.MemberForm{
			declaration.MemberNamed,
			declaration.MemberContained,
			declaration.MemberSelector,
		},
	}
}

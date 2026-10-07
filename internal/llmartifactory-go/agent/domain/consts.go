package domain

import (
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
)

const (
	AgentArtifactKind artifactModel.ArtifactKind = artifactModel.ArtifactKind(
		agentv1.AgentType,
	)

	ManagedAgentPackageKind       managedpackageModel.PackageKind = "agent"
	BuiltinAgentPluginPackageKind managedpackageModel.PackageKind = "agent-plugin"
	AgentSchemaID                 schemaModel.SchemaID            = agentv1.AgentSchemaID

	AgentSchemaVersion     = agentv1.AgentSchemaVersion
	BuiltInInstallerName   = "agent.agent"
	HydrationSchemaVersion = "agent.agent.builtin-hydration/v1"
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

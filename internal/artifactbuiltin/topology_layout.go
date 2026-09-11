package artifactbuiltin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	ApplicationDataDirectoryName = "flexigpt"

	SettingsDirectoryName         = "settings_v1"
	ConversationsDirectoryName    = "conversations_v1"
	ModelPresetsDirectoryName     = "model_presets_v1"
	ToolsDirectoryName            = "tools_v1"
	AssistantPresetsDirectoryName = "assistant_presets_v1"
	ArtifactStoreDirectoryName    = "artifacts_v1"

	ApplicationDirectoryMode = 0o700

	ArtifactStoreManifestFileName      = "store.json"
	ArtifactStoreMetadataFileName      = "app.sqlite"
	ArtifactStoreContentDirectoryName  = "content"
	ArtifactStoreStagingDirectoryName  = "staging"
	ArtifactStoreManifestTemporaryName = "store.json.tmp-"

	ArtifactStoreFormat        = "flexigpt-artifactstore/v1"
	ArtifactStoreContentLayout = "semantic-packages/v1"

	ArtifactStoreDirectoryMode = 0o750
	ArtifactStoreManifestMode  = 0o600

	ManagedPackageTemporaryPrefix = "package-"
	ManagedPackagePreviousPrefix  = "previous-package-"
	ManagedPackageRemovalPrefix   = "remove-"

	UnversionedPackageVersion basespec.LogicalVersion = "unversioned"

	AgentSkillPackageKind  source.PackageKind = "agent.skill"
	SkillBundlePackageKind source.PackageKind = "skill.bundle"
	MCPBundlePackageKind   source.PackageKind = "mcp.bundle"

	AgentSkillDefinitionFileName basespec.Locator = "SKILL.md"
	SkillCollectionFileName      basespec.Locator = "collection.json"
	MCPBundleDocumentFileName    basespec.Locator = "mcps.json"

	RepositoryRootLocator       basespec.Locator = "."
	WorkspaceDescriptorFileName basespec.Locator = "workspace.json"
	WorkspaceSkillRootLocator   basespec.Locator = "skills"
	WorkspaceMarkdownPattern                     = "*.md"

	WorkspaceAgentsFileName basespec.Locator = "AGENTS.md"
	WorkspaceClaudeFileName basespec.Locator = "CLAUDE.md"
	WorkspaceReadmeFileName basespec.Locator = "README.md"

	EmbeddedSkillRegistryFileName                  = "skill-registry.json"
	EmbeddedMCPRegistryFileName                    = "mcp_artifact_registry.json"
	EmbeddedSkillDataRoot         basespec.Locator = "skills"
	EmbeddedMCPDataRoot           basespec.Locator = "mcps"
	EmbeddedSkillRegistryLocator  basespec.Locator = "skills/skill-registry.json"
	EmbeddedMCPRegistryLocator    basespec.Locator = "mcps/mcp_artifact_registry.json"

	ExternalGitMetadataDirectoryName = ".git"
)

const (
	BuiltinRootID          root.RootID         = "0192c4c0-0000-7000-8000-000000000001"
	BuiltinRootStorageKey  basespec.StorageKey = "builtins"
	BuiltinRootDisplayName                     = "Application Built-ins"
	BuiltinRootDescription                     = "Protected application-provided portable artifact packages."

	BuiltinSourceID          source.SourceID     = "0192c4c0-0001-7000-8000-000000000001"
	BuiltinSourceStorageKey  basespec.StorageKey = "catalog"
	BuiltinSourceDisplayName                     = "Application Built-in Packages"

	WorkspaceRootID          root.RootID         = "0198f097-0d5b-7000-8000-000000000001"
	WorkspaceRootStorageKey  basespec.StorageKey = "workspaces"
	WorkspaceRootDisplayName                     = "FlexiGPT Workspaces"
	WorkspaceRootDescription                     = "Local namespace for user Workspace collections."

	MCPUserRootID          root.RootID         = "0198f097-0d5b-7000-8000-000000000002"
	MCPUserRootStorageKey  basespec.StorageKey = "mcp"
	MCPUserRootDisplayName                     = "FlexiGPT MCP Bundles"
	MCPUserRootDescription                     = "Local namespace for user-managed MCP Bundles."

	DefaultMCPBundleCollectionID collection.CollectionID = "0198f097-0d5b-7000-8000-000000000020"
	DefaultMCPBundleSourceID     source.SourceID         = "0198f097-0d5b-7000-8000-000000000021"
	DefaultMCPBundleSourceKey    basespec.StorageKey     = "base"
	DefaultMCPBundleLogicalName                          = "base"
	DefaultMCPBundleDisplayName                          = "Base MCP Servers"
	DefaultMCPBundleDescription                          = "Editable starter bundle for user-managed MCP server definitions."
)

const (
	MCPHostName    = "FlexiGPT"
	MCPHostVersion = "dev"

	MCPBuiltInInstallerName = "mcp.bundle"

	MCPBundleHydrationFingerprintSchemaVersion = "mcp.builtin-hydration/v1"

	MCPSchemaVersion = "v1"

	MCPServerSubresourceDirectory basespec.SubresourceLocator = "mcpServers"
	MCPPolicySubresourceDirectory basespec.SubresourceLocator = "policies"

	DecoderRevision                    = "mcp.bundle.discovery.v1"
	DecoderID       basespec.DecoderID = "mcp.bundle-json"
)

type (
	WorkspaceContextRole       string
	WorkspaceContextMediaType  string
	WorkspaceContextPreference string
)

const (
	ManagedAttachmentRole collection.AttachmentRole = "managed"
	BuiltInAttachmentRole collection.AttachmentRole = "builtin"

	AgentSkillArtifactKind   artifact.ArtifactKind = "agent.skill"
	AgentSkillSchemaID       schema.SchemaID       = "agent.skill.v1"
	AgentSkillDecoderID      basespec.DecoderID    = "agent.skill-markdown"
	AgentSkillSchemaVersion                        = "v1"
	AgentSkillInsertLabelKey                       = "skill.insert"

	// AgentSkillBuiltInInstallerName is stable across built-in hydration
	// revisions so prior hydration records remain discoverable.
	AgentSkillBuiltInInstallerName = "agent.skill"

	AgentSkillHydrationFingerprintSchemaVersion = "agent.skill.builtin-hydration/v1"

	WorkspaceContextArtifactKind  artifact.ArtifactKind = "workspace.context"
	WorkspaceContextSchemaID      schema.SchemaID       = "workspace.context.v1"
	WorkspaceContextDecoderID     basespec.DecoderID    = "workspace.context-markdown"
	WorkspaceContextSchemaVersion                       = "v1"
	WorkspaceContextRoleLabelKey                        = "context.role"

	WorkspaceContextRoleAgentInstructions     WorkspaceContextRole = "agent-instructions"
	WorkspaceContextRoleAssistantInstructions WorkspaceContextRole = "assistant-instructions"
	WorkspaceContextRoleProjectReadme         WorkspaceContextRole = "project-readme"
	WorkspaceContextRoleProjectContext        WorkspaceContextRole = "project-context"

	WorkspaceContextMediaTypeMarkdown WorkspaceContextMediaType = "text/markdown"

	WorkspaceContextPreferenceIncludeReadme WorkspaceContextPreference = "include-readme"
)

type WorkspaceContextFileConvention struct {
	FileName         basespec.Locator           `json:"fileName"`
	Role             WorkspaceContextRole       `json:"role"`
	DefaultDiscovery bool                       `json:"defaultDiscovery"`
	Preference       WorkspaceContextPreference `json:"preference,omitempty"`
	RuntimeOrder     int                        `json:"runtimeOrder"`
}

var externalTraversalExcludedDirectoryNames = []string{
	".git",
	".hg",
	".svn",
	"node_modules",
	"vendor",
	"bower_components",
}

var workspaceContextFileConventions = []WorkspaceContextFileConvention{
	{
		FileName:         WorkspaceAgentsFileName,
		Role:             WorkspaceContextRoleAgentInstructions,
		DefaultDiscovery: true,
		RuntimeOrder:     100,
	},
	{
		FileName:         WorkspaceClaudeFileName,
		Role:             WorkspaceContextRoleAssistantInstructions,
		DefaultDiscovery: true,
		RuntimeOrder:     200,
	},
	{
		FileName:     WorkspaceReadmeFileName,
		Role:         WorkspaceContextRoleProjectReadme,
		Preference:   WorkspaceContextPreferenceIncludeReadme,
		RuntimeOrder: 300,
	},
}

func ApplicationStorageNames() []string {
	return []string{
		ApplicationDataDirectoryName,
		SettingsDirectoryName,
		ConversationsDirectoryName,
		ModelPresetsDirectoryName,
		ToolsDirectoryName,
		AssistantPresetsDirectoryName,
		ArtifactStoreDirectoryName,
		ArtifactStoreManifestFileName,
		ArtifactStoreMetadataFileName,
		ArtifactStoreContentDirectoryName,
		ArtifactStoreStagingDirectoryName,
		ArtifactStoreManifestTemporaryName,
		string(WorkspaceDescriptorFileName),
		string(WorkspaceSkillRootLocator),
		string(AgentSkillDefinitionFileName),
		string(SkillCollectionFileName),
		string(MCPBundleDocumentFileName),
	}
}

func ExternalTraversalExcludedDirectoryNames() []string {
	return append(
		[]string(nil),
		externalTraversalExcludedDirectoryNames...,
	)
}

func WorkspaceContextFileConventions() []WorkspaceContextFileConvention {
	return append(
		[]WorkspaceContextFileConvention(nil),
		workspaceContextFileConventions...,
	)
}

func WorkspaceSkillRoots() []basespec.Locator {
	return []basespec.Locator{WorkspaceSkillRootLocator}
}

func BuiltinTopologyDeclaration() topology.Declaration {
	return topology.Declaration{
		Root: root.RootDraft{
			ID:          BuiltinRootID,
			StorageKey:  BuiltinRootStorageKey,
			DisplayName: BuiltinRootDisplayName,
			Description: BuiltinRootDescription,
		},
		Sources: []source.Draft{{
			ID:          BuiltinSourceID,
			StorageKey:  BuiltinSourceStorageKey,
			Kind:        source.SourceKindManagedDirectory,
			DisplayName: BuiltinSourceDisplayName,
			Enabled:     true,
			Config:      json.RawMessage(jsonutil.EmptyObject),
		}},
	}
}

func RetainedRootDrafts() []root.RootDraft {
	return []root.RootDraft{
		{
			ID:          WorkspaceRootID,
			StorageKey:  WorkspaceRootStorageKey,
			DisplayName: WorkspaceRootDisplayName,
			Description: WorkspaceRootDescription,
		},
		{
			ID:          MCPUserRootID,
			StorageKey:  MCPUserRootStorageKey,
			DisplayName: MCPUserRootDisplayName,
			Description: MCPUserRootDescription,
		},
	}
}

func ProtectedRootIDs() []root.RootID {
	return []root.RootID{BuiltinRootID}
}

func RetainedRootIDs() []root.RootID {
	return []root.RootID{
		WorkspaceRootID,
		MCPUserRootID,
	}
}

func ValidateApplicationTopology() error {
	for _, name := range ApplicationStorageNames() {
		if name == "" || strings.HasPrefix(name, ".") {
			return fmt.Errorf(
				"%w: application storage name %q is invalid",
				basespec.ErrInvalid,
				name,
			)
		}
	}

	declaration := BuiltinTopologyDeclaration()
	if err := declaration.Validate(); err != nil {
		return err
	}
	if len(declaration.Sources) != 1 {
		return fmt.Errorf(
			"%w: built-in topology must declare exactly one source",
			basespec.ErrInvalid,
		)
	}

	seenRootIDs := map[root.RootID]struct{}{
		declaration.Root.ID: {},
	}
	seenStorageKeys := map[basespec.StorageKey]struct{}{
		declaration.Root.StorageKey: {},
	}
	for _, draft := range RetainedRootDrafts() {
		if err := draft.ID.Validate(); err != nil {
			return err
		}
		if err := draft.StorageKey.Validate(); err != nil {
			return err
		}
		if _, exists := seenRootIDs[draft.ID]; exists {
			return fmt.Errorf(
				"%w: duplicate application root ID %q",
				basespec.ErrConflict,
				draft.ID,
			)
		}
		if _, exists := seenStorageKeys[draft.StorageKey]; exists {
			return fmt.Errorf(
				"%w: duplicate application root storage key %q",
				basespec.ErrConflict,
				draft.StorageKey,
			)
		}
		seenRootIDs[draft.ID] = struct{}{}
		seenStorageKeys[draft.StorageKey] = struct{}{}
	}

	if declaration.Root.ID == root.RootID(declaration.Sources[0].ID) {
		return fmt.Errorf(
			"%w: built-in root and source IDs must differ",
			basespec.ErrConflict,
		)
	}
	return nil
}

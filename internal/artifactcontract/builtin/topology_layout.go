package builtin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
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
	ArtifactStoreDirectoryName    = "artifacts_v3"

	ApplicationDirectoryMode = 0o700

	ArtifactStoreManifestFileName      = "store.json"
	ArtifactStoreMetadataFileName      = "app.sqlite"
	ArtifactStoreContentDirectoryName  = "content"
	ArtifactStoreStagingDirectoryName  = "staging"
	ArtifactStoreManifestTemporaryName = "store.json.tmp-"

	ArtifactStoreFormat        = "flexigpt-artifactstore/v3"
	ArtifactStoreContentLayout = "source-packages/v1"

	ArtifactStoreDirectoryMode = 0o750
	ArtifactStoreManifestMode  = 0o600

	ManagedPackageTemporaryPrefix = "package-"
	ManagedPackagePreviousPrefix  = "previous-package-"
	ManagedPackageRemovalPrefix   = "remove-"

	UnversionedPackageVersion basespec.LogicalVersion = "unversioned"

	RepositoryRootLocator basespec.Locator = "."

	ExternalGitMetadataDirectoryName = ".git"

	EmbeddedSkillDataRoot basespec.Locator = "skills"
	EmbeddedMCPDataRoot   basespec.Locator = "mcps"
)

const (
	BuiltinRootID          root.RootID         = "0192c4c0-0000-7000-8000-000000000001"
	BuiltinRootStorageKey  basespec.StorageKey = "builtins"
	BuiltinRootDisplayName                     = "Application Built-ins"
	BuiltinRootDescription                     = "Protected application-provided artifact source namespace."

	BuiltinSourceID          source.SourceID     = "0192c4c0-0001-7000-8000-000000000001"
	BuiltinSourceStorageKey  basespec.StorageKey = "artifacts"
	BuiltinSourceDisplayName                     = "Application Built-in Artifact Source"
)

const (
	MCPHostName    = "FlexiGPT"
	MCPHostVersion = "dev"
)

var externalTraversalExcludedDirectoryNames = []string{
	".git",
	".hg",
	".svn",
	"node_modules",
	"vendor",
	"bower_components",
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
	}
}

func ExternalTraversalExcludedDirectoryNames() []string {
	return append(
		[]string(nil),
		externalTraversalExcludedDirectoryNames...,
	)
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
			// Built-in collections are grouping declarations only. Skills,
			// MCP servers, and MCP policies are independently discovered
			// Artifacts in the same protected Source.
			Discovery: source.DiscoverySpec{
				DirectoryRoots: []source.DirectoryRoot{{
					Root:      RepositoryRootLocator,
					Recursive: true,
					IncludePatterns: []string{
						"**/collection.yaml",
						"**/SKILL.md",
						"**/declarations/**/*.json",
					},
				}},
				Authoritative: true,
			},
		}},
	}
}

func RetainedRootDrafts() []root.RootDraft {
	return nil
}

func ProtectedRootIDs() []root.RootID {
	return []root.RootID{BuiltinRootID}
}

func RetainedRootIDs() []root.RootID {
	return nil
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
			"%w: built-in topology must declare exactly one Source",
			basespec.ErrInvalid,
		)
	}
	if declaration.Root.ID == root.RootID(
		declaration.Sources[0].ID,
	) {
		return fmt.Errorf(
			"%w: built-in Root and Source IDs must differ",
			basespec.ErrConflict,
		)
	}
	return nil
}

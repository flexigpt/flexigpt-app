package workspace

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
)

type discoveryProfile struct {
	explicitLocators []artifactstore.Locator
	directoryRoots   []discovery.DirectoryRoot
}

type attachmentOperation struct {
	role                                 artifactstore.AttachmentRole
	canAttach                            bool
	isPrimary                            bool
	requiredSourceKind                   artifactstore.SourceKind
	defaultPriority                      int
	defaultAuthoritative                 bool
	includeReadmeWhenRequested           bool
	appliesWorkspaceDiscoveryPreferences bool
	allowsAttachmentDiscoveryOverrides   bool
	profile                              discoveryProfile
}

var primaryDiscoveryProfile = discoveryProfile{
	explicitLocators: []artifactstore.Locator{
		DefinitionLocator,
		ContextAgentsLocator,
		ContextClaudeLocator,
	},
	directoryRoots: []discovery.DirectoryRoot{
		{
			Root:      WorkspaceMetadataLocator,
			Recursive: true,
			IncludePatterns: []string{
				jsonFilePattern,
				yamlFilePattern,
				yamlShortFilePattern,
				markdownFilePattern,
			},
		},
		{
			Root:      WorkspaceSkillsLocator,
			Recursive: true,
			IncludePatterns: []string{
				skillDefinitionFileName,
			},
		},
	},
}

var attachedDiscoveryProfile = discoveryProfile{
	directoryRoots: []discovery.DirectoryRoot{
		{
			Root:      RepositoryRootLocator,
			Recursive: true,
			IncludePatterns: []string{
				jsonFilePattern,
				yamlFilePattern,
				yamlShortFilePattern,
				markdownFilePattern,
			},
		},
	},
}

// attachmentOperationMatrix is the workspace attachment lifecycle and
// discovery-operation matrix.
//
// A role must be present here before it can be attached, validated, or planned.
var attachmentOperationMatrix = [...]attachmentOperation{
	{
		role:                                 RolePrimary,
		isPrimary:                            true,
		requiredSourceKind:                   fsdir.Kind,
		defaultPriority:                      PrimaryPriority,
		defaultAuthoritative:                 true,
		includeReadmeWhenRequested:           true,
		appliesWorkspaceDiscoveryPreferences: true,
		profile:                              primaryDiscoveryProfile,
	},
	{
		role:                               RoleBuiltIn,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
		profile:                            attachedDiscoveryProfile,
	},
	{
		role:                               RoleLibrary,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
		profile:                            attachedDiscoveryProfile,
	},
	{
		role:                               RoleAttachedPackage,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
		profile:                            attachedDiscoveryProfile,
	},
	{
		role:                               RoleOverlay,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
		profile:                            attachedDiscoveryProfile,
	},
}

func attachmentOperationFor(
	role artifactstore.AttachmentRole,
) (attachmentOperation, bool) {
	for _, operation := range attachmentOperationMatrix {
		if operation.role == role {
			return operation, true
		}
	}
	return attachmentOperation{}, false
}

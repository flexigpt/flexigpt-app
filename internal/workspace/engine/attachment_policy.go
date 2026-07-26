package engine

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
)

func DefaultDiscoveryProfiles() DiscoveryProfiles {
	return DiscoveryProfiles{
		Primary: DiscoveryProfile{},
		Attached: DiscoveryProfile{
			DirectoryRoots: []discovery.DirectoryRoot{
				{
					Root:      RepositoryRootLocator,
					Recursive: true,
					IncludePatterns: []string{
						markdownFilePattern,
					},
				},
			},
		},
	}
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
		defaultAuthoritative:                 true,
		includeReadmeWhenRequested:           true,
		appliesWorkspaceDiscoveryPreferences: true,
	},
	{
		role:                               RoleBuiltIn,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
	},
	{
		role:                               RoleLibrary,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
	},
	{
		role:                               RoleAttachedPackage,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
	},
	{
		role:                               RoleOverlay,
		canAttach:                          true,
		defaultAuthoritative:               true,
		allowsAttachmentDiscoveryOverrides: true,
	},
}

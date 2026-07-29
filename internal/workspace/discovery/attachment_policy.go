package discovery

import (
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const markdownFilePattern = "*.md"

func DefaultDiscoveryProfiles() spec.DiscoveryProfiles {
	return spec.DiscoveryProfiles{
		Primary: spec.DiscoveryProfile{},
		Attached: spec.DiscoveryProfile{
			DirectoryRoots: []spec.DirectoryRoot{
				{
					Root:      spec.RepositoryRootLocator,
					Recursive: true,
					IncludePatterns: []string{
						markdownFilePattern,
					},
				},
			},
		},
	}
}

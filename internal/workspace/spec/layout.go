package spec

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"

const (
	WorkspaceMetadataDirectory                  = ".flexigpt"
	WorkspaceMetadataLocator   basespec.Locator = WorkspaceMetadataDirectory
	RepositoryRootLocator      basespec.Locator = "."

	WorkspaceDescriptorFileName = "workspace.json"
	DescriptorLocator           = WorkspaceMetadataDirectory + "/" + WorkspaceDescriptorFileName

	DefaultMarkdownIncludePattern = "*.md"
)

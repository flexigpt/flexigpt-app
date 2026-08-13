package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const (
	RepositoryRootLocator basespec.Locator = artifactbuiltin.RepositoryRootLocator

	WorkspaceDescriptorFileName basespec.Locator = artifactbuiltin.WorkspaceDescriptorFileName

	DescriptorLocator basespec.Locator = artifactbuiltin.WorkspaceDescriptorLocator

	DefaultMarkdownIncludePattern = artifactbuiltin.WorkspaceMarkdownPattern
)

package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

const (
	RepositoryRootLocator basespec.Locator = builtinSchema.RepositoryRootLocator

	WorkspaceDescriptorFileName basespec.Locator = builtinSchema.WorkspaceDescriptorFileName

	DescriptorLocator basespec.Locator = builtinSchema.WorkspaceDescriptorLocator

	DefaultMarkdownIncludePattern = builtinSchema.WorkspaceMarkdownPattern
)

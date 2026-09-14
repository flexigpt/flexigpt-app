package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type WorkspaceRef = workspaceDomain.WorkspaceRef

type FilesystemSourceRegistration struct {
	RootID            root.RootID `json:"rootID"`
	RootPath          string      `json:"rootPath"`
	SourceDisplayName string      `json:"sourceDisplayName"`
}

type WorkspaceLoad struct {
	Workspace workspaceDomain.Workspace
	Roots     []artifactcontract.Entry
}

type WorkspaceRefresh struct {
	Workspace WorkspaceRef
	Result    refresh.RefreshRootResult
}

type WorkspaceArtifactSettings struct {
	RuntimeDisabled bool `json:"runtimeDisabled"`
}

type WorkspaceArtifactView struct {
	Artifact           artifact.ArtifactRef        `json:"artifact"`
	Revision           uint64                      `json:"revision"`
	DisplayName        string                      `json:"displayName"`
	Kind               artifact.ArtifactKind       `json:"kind"`
	LogicalName        basespec.LogicalName        `json:"logicalName"`
	LogicalVersion     basespec.LogicalVersion     `json:"logicalVersion,omitempty"`
	Enabled            bool                        `json:"enabled"`
	State              artifact.State              `json:"state"`
	SourceID           source.SourceID             `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
	RuntimeDisabled    bool                        `json:"runtimeDisabled"`
}

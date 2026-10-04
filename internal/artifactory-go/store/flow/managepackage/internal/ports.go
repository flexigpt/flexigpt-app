package internal

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

type ArtifactCommands interface {
	Get(ctx context.Context, ref artifactModel.ArtifactRef) (artifactModel.Artifact, error)
	FindByOrigin(
		ctx context.Context,
		rootID rootModel.RootID,
		s artifactModel.SourceBinding,
		k artifactModel.ArtifactKind,
	) (artifactModel.Artifact, error)
}

type SourceRunner interface {
	RefreshSource(
		ctx context.Context,
		rootID rootModel.RootID,
		s sourceModel.SourceID,
	) (refreshModel.RefreshSourceResult, error)
	InspectSource(ctx context.Context, rootID rootModel.RootID, s sourceModel.SourceID) (refreshModel.Inspection, error)
}

type Dependencies struct {
	Artifacts       ArtifactCommands
	Refresh         SourceRunner
	Sources         source.API
	Runtime         source.Runtime
	ContentMutation source.ContentMutation
	Packages        managedpackage.Runtime
	Policy          root.Policy
}

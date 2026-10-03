package refresh

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

// CompiledDocumentRegistrar is the trusted installation-only refresh
// capability used to register generated built-in declaration witnesses.
type CompiledDocumentRegistrar interface {
	RegisterCompiledDocuments(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		packages []installModel.CompiledPackage,
	) error
}

type StateReader interface {
	GetRefreshState(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.State, error)
}

type ArtifactReader interface {
	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) ([]artifactModel.Artifact, error)
}

type Repository interface {
	Publish(
		ctx context.Context,
		publication Publication,
	) (refreshModel.State, error)
}

type API interface {
	RefreshRoot(
		ctx context.Context,
		rootID rootModel.RootID,
	) (refreshModel.RefreshRootResult, error)

	RefreshSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.RefreshSourceResult, error)

	// Inspection is refresh-flow state and is represented by refresh/model.
	InspectSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.Inspection, error)
}

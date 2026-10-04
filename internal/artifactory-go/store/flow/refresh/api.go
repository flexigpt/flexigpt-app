package refresh

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// CompiledDocumentRegistrar accepts Ingest-owned evidence. Install translates
// package plans before invoking this trusted registration capability.
type CompiledDocumentRegistrar interface {
	RegisterCompiledDocuments(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		documents []ingest.CompiledDocument,
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

// ArtifactSynchronizer is the exact Artifact capability Refresh needs for
// ordinary observations and explicit lifecycle invalidation.
type ArtifactSynchronizer interface {
	Synchronize(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceValue sourceModel.Source,
		observations []ingest.Observation,
		seenLocators []spec.Locator,
		existing []artifactModel.Artifact,
	) (artifact.Synchronization, error)
	DeriveLifecycleInvalidation(
		ctx context.Context,
		transition source.LifecycleTransition,
		existing []artifactModel.Artifact,
	) ([]artifact.SourceStateUpdate, error)
}

// Repository executes Refresh's explicit atomic publication commands.
type Repository interface {
	Publish(ctx context.Context, publication Publication) (refreshModel.State, error)
	PublishLifecycle(ctx context.Context, publication LifecyclePublication) error
}

// MetadataInspector is the narrow Resource dependency for metadata freshness
// checks without exposing refresh mutation APIs.
type MetadataInspector interface {
	InspectSourceMetadata(ctx context.Context, source sourceModel.Source) (refreshModel.Inspection, error)
}

type API interface {
	RefreshRoot(ctx context.Context, rootID rootModel.RootID) (refreshModel.RefreshRootResult, error)
	RefreshSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.RefreshSourceResult, error)
	InspectSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.Inspection, error)
}

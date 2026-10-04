package artifact

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Reader loads complete Artifact entities. Catalog persistence is deliberately
// separate because catalog queries return lightweight committed projections.
type Reader interface {
	Get(ctx context.Context, ref artifactModel.ArtifactRef) (artifactModel.Artifact, error)
	GetMany(ctx context.Context, refs []artifactModel.ArtifactRef) ([]artifactModel.Artifact, error)
	ListByRoot(ctx context.Context, rootID rootModel.RootID) ([]artifactModel.Artifact, error)
	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) ([]artifactModel.Artifact, error)
	FindByIdentity(
		ctx context.Context,
		rootID rootModel.RootID,
		kind artifactModel.ArtifactKind,
		logicalName spec.LogicalName,
	) ([]artifactModel.Artifact, error)
	FindByOrigin(
		ctx context.Context,
		rootID rootModel.RootID,
		binding artifactModel.SourceBinding,
		kind artifactModel.ArtifactKind,
	) (artifactModel.Artifact, error)
}

type DefinitionReader = definition.API

// Repository persists complete Artifact entities and source-owned state.
type Repository interface {
	Reader
	Create(ctx context.Context, value artifactModel.Artifact) error
	UpdateLocal(ctx context.Context, value artifactModel.Artifact, expectedRevision uint64) error
	UpdateSourceState(ctx context.Context, value SourceStateUpdate) error
	Purge(ctx context.Context, ref artifactModel.ArtifactRef, expectedRevision uint64) error
}

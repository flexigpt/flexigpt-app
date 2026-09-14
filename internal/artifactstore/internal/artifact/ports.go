package artifactimpl

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	ListByRoot(
		ctx context.Context,
		rootID root.RootID,
	) ([]artifact.Artifact, error)

	ListBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) ([]artifact.Artifact, error)

	FindByIdentity(
		ctx context.Context,
		rootID root.RootID,
		kind artifact.ArtifactKind,
		logicalName basespec.LogicalName,
	) ([]artifact.Artifact, error)

	FindByOrigin(
		ctx context.Context,
		rootID root.RootID,
		binding artifact.SourceBinding,
		kind artifact.ArtifactKind,
	) (artifact.Artifact, error)
}

type Repository interface {
	Reader

	Create(
		ctx context.Context,
		value artifact.Artifact,
	) error

	UpdateLocal(
		ctx context.Context,
		value artifact.Artifact,
		expectedRevision uint64,
	) error

	UpdateSourceState(
		ctx context.Context,
		value SourceStateUpdate,
	) error

	Purge(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error
}

type DefinitionReader interface {
	GetDefinition(
		ctx context.Context,
		rootID root.RootID,
		digest cryptoutil.Digest,
	) (definition.Definition, error)
}

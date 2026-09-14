package sqlite

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifact"
	refreshimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/refresh"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type SourceRepository struct {
	store *Store
}

type RootRepository struct {
	store *Store
}

type ArtifactRepository struct {
	store *Store
}

type DefinitionRepository struct {
	store *Store
}

type RefreshStateRepository struct {
	store *Store
}

func (s *Store) Sources() *SourceRepository {
	return &SourceRepository{store: s}
}

func (s *Store) Roots() *RootRepository {
	return &RootRepository{store: s}
}

func (s *Store) Artifacts() *ArtifactRepository {
	return &ArtifactRepository{store: s}
}

func (s *Store) Definitions() *DefinitionRepository {
	return &DefinitionRepository{store: s}
}

func (s *Store) RefreshStates() *RefreshStateRepository {
	return &RefreshStateRepository{store: s}
}

func (s *Store) Publisher() *Publisher {
	return &Publisher{store: s}
}

func (r *SourceRepository) Create(
	ctx context.Context,
	value source.Source,
) error {
	return r.store.createSource(ctx, value)
}

func (r *SourceRepository) Get(
	ctx context.Context,
	rootID root.RootID,
	id source.SourceID,
) (source.Source, error) {
	return r.store.getSource(ctx, rootID, id)
}

func (r *SourceRepository) List(
	ctx context.Context,
	rootID root.RootID,
) ([]source.Source, error) {
	return r.store.listSources(ctx, rootID)
}

func (r *SourceRepository) Update(
	ctx context.Context,
	value source.Source,
	expectedRevision uint64,
) error {
	return r.store.updateSource(ctx, value, expectedRevision)
}

func (r *SourceRepository) Retire(
	ctx context.Context,
	value source.Source,
	expectedRevision uint64,
) error {
	return r.store.retireSource(ctx, value, expectedRevision)
}

func (r *SourceRepository) Discard(
	ctx context.Context,
	rootID root.RootID,
	id source.SourceID,
	expectedRevision uint64,
) error {
	return r.store.discardSource(
		ctx,
		rootID,
		id,
		expectedRevision,
	)
}

func (r *SourceRepository) Purge(
	ctx context.Context,
	rootID root.RootID,
	id source.SourceID,
	expectedRevision uint64,
) error {
	return r.store.purgeSource(
		ctx,
		rootID,
		id,
		expectedRevision,
	)
}

func (r *RootRepository) Create(
	ctx context.Context,
	value root.Root,
) error {
	return r.store.createRoot(ctx, value)
}

func (r *RootRepository) Get(
	ctx context.Context,
	id root.RootID,
) (root.Root, error) {
	return r.store.getRoot(ctx, id)
}

func (r *RootRepository) List(
	ctx context.Context,
) ([]root.Root, error) {
	return r.store.listRoots(ctx)
}

func (r *RootRepository) Update(
	ctx context.Context,
	value root.Root,
	expectedRevision uint64,
) error {
	return r.store.updateRoot(ctx, value, expectedRevision)
}

func (r *RootRepository) Retire(
	ctx context.Context,
	value root.Root,
	expectedRevision uint64,
) error {
	return r.store.retireRoot(ctx, value, expectedRevision)
}

func (r *RootRepository) Purge(
	ctx context.Context,
	id root.RootID,
	expectedRevision uint64,
) error {
	return r.store.purgeRoot(ctx, id, expectedRevision)
}

func (r *ArtifactRepository) Get(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	return r.store.getArtifact(ctx, ref)
}

func (r *ArtifactRepository) ListByRoot(
	ctx context.Context,
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return r.store.listArtifactsByRoot(ctx, rootID)
}

func (r *ArtifactRepository) ListBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	return r.store.listArtifactsBySource(
		ctx,
		rootID,
		sourceID,
	)
}

func (r *ArtifactRepository) FindByIdentity(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
	logicalName basespec.LogicalName,
) ([]artifact.Artifact, error) {
	return r.store.findArtifactsByIdentity(
		ctx,
		rootID,
		kind,
		logicalName,
	)
}

func (r *ArtifactRepository) FindByOrigin(
	ctx context.Context,
	rootID root.RootID,
	binding artifact.SourceBinding,
	kind artifact.ArtifactKind,
) (artifact.Artifact, error) {
	return r.store.findArtifactByOrigin(
		ctx,
		rootID,
		binding,
		kind,
	)
}

func (r *ArtifactRepository) Create(
	ctx context.Context,
	value artifact.Artifact,
) error {
	return r.store.createArtifact(ctx, value)
}

func (r *ArtifactRepository) UpdateLocal(
	ctx context.Context,
	value artifact.Artifact,
	expectedRevision uint64,
) error {
	return r.store.updateArtifactLocal(
		ctx,
		value,
		expectedRevision,
	)
}

func (r *ArtifactRepository) UpdateSourceState(
	ctx context.Context,
	value artifactimpl.SourceStateUpdate,
) error {
	return r.store.updateArtifactSourceState(ctx, value)
}

func (r *ArtifactRepository) Purge(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return r.store.purgeArtifact(
		ctx,
		ref,
		expectedRevision,
	)
}

func (r *DefinitionRepository) GetDefinition(
	ctx context.Context,
	rootID root.RootID,
	digest cryptoutil.Digest,
) (definition.Definition, error) {
	return r.store.getDefinition(ctx, rootID, digest)
}

func (r *RefreshStateRepository) GetRefreshState(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.RefreshState, error) {
	return r.store.getRefreshState(ctx, rootID, sourceID)
}

var (
	_ artifactimpl.Repository        = (*ArtifactRepository)(nil)
	_ artifactimpl.DefinitionReader  = (*DefinitionRepository)(nil)
	_ refreshimpl.RefreshStateReader = (*RefreshStateRepository)(nil)
)

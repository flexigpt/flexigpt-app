package sqlite

import (
	"context"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/impl"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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

func (s *Store) LocalState() *LocalStateRepository { return &LocalStateRepository{store: s} }

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
	value sourceModel.Source,
) error {
	return r.store.createSource(ctx, value)
}

func (r *SourceRepository) Get(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
) (sourceModel.Source, error) {
	return r.store.getSource(ctx, rootID, id)
}

func (r *SourceRepository) FindByStorageKey(
	ctx context.Context,
	rootID rootModel.RootID,
	storageKey spec.StorageKey,
) (sourceModel.Source, error) {
	return r.store.findSourceByStorageKey(ctx, rootID, storageKey)
}

func (r *SourceRepository) List(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]sourceModel.Source, error) {
	return r.store.listSources(ctx, rootID)
}

func (r *SourceRepository) Update(
	ctx context.Context,
	value sourceModel.Source,
	expectedRevision uint64,
) error {
	return r.store.updateSource(ctx, value, expectedRevision)
}

func (r *SourceRepository) Retire(
	ctx context.Context,
	value sourceModel.Source,
	expectedRevision uint64,
) error {
	return r.store.retireSource(ctx, value, expectedRevision)
}

func (r *SourceRepository) Discard(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
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
	rootID rootModel.RootID,
	id sourceModel.SourceID,
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
	value rootModel.Root,
) error {
	return r.store.createRoot(ctx, value)
}

func (r *RootRepository) Get(
	ctx context.Context,
	id rootModel.RootID,
) (rootModel.Root, error) {
	return r.store.getRoot(ctx, id)
}

func (r *RootRepository) List(
	ctx context.Context,
) ([]rootModel.Root, error) {
	return r.store.listRoots(ctx)
}

func (r *RootRepository) Update(
	ctx context.Context,
	value rootModel.Root,
	expectedRevision uint64,
) error {
	return r.store.updateRoot(ctx, value, expectedRevision)
}

func (r *RootRepository) Retire(
	ctx context.Context,
	value rootModel.Root,
	expectedRevision uint64,
) error {
	return r.store.retireRoot(ctx, value, expectedRevision)
}

func (r *RootRepository) Purge(
	ctx context.Context,
	id rootModel.RootID,
	expectedRevision uint64,
) error {
	return r.store.purgeRoot(ctx, id, expectedRevision)
}

func (r *ArtifactRepository) Get(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	return r.store.getArtifact(ctx, ref)
}

func (r *ArtifactRepository) ListByRoot(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]artifactModel.Artifact, error) {
	return r.store.listArtifactsByRoot(ctx, rootID)
}

func (r *ArtifactRepository) ListBySource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) ([]artifactModel.Artifact, error) {
	return r.store.listArtifactsBySource(
		ctx,
		rootID,
		sourceID,
	)
}

func (r *ArtifactRepository) ListCatalogByRoot(
	ctx context.Context,
	rootID rootModel.RootID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return r.store.listArtifactCatalogByRoot(ctx, rootID, options)
}

func (r *ArtifactRepository) ListCatalogBySource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return r.store.listArtifactCatalogBySource(
		ctx,
		rootID,
		sourceID,
		options,
	)
}

func (r *ArtifactRepository) FindCatalogByIdentity(
	ctx context.Context,
	rootID rootModel.RootID,
	kind artifactModel.ArtifactKind,
	logicalName spec.LogicalName,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return r.store.findArtifactCatalogByIdentity(
		ctx,
		rootID,
		kind,
		logicalName,
		options,
	)
}

func (r *ArtifactRepository) GetMany(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) ([]artifactModel.Artifact, error) {
	return r.store.getArtifactsByReferences(ctx, refs)
}

func (r *ArtifactRepository) FindByIdentity(
	ctx context.Context,
	rootID rootModel.RootID,
	kind artifactModel.ArtifactKind,
	logicalName spec.LogicalName,
) ([]artifactModel.Artifact, error) {
	return r.store.findArtifactsByIdentity(
		ctx,
		rootID,
		kind,
		logicalName,
	)
}

func (r *ArtifactRepository) FindByOrigin(
	ctx context.Context,
	rootID rootModel.RootID,
	binding artifactModel.SourceBinding,
	kind artifactModel.ArtifactKind,
) (artifactModel.Artifact, error) {
	return r.store.findArtifactByOrigin(
		ctx,
		rootID,
		binding,
		kind,
	)
}

func (r *ArtifactRepository) Create(
	ctx context.Context,
	value artifactModel.Artifact,
) error {
	return r.store.createArtifact(ctx, value)
}

func (r *ArtifactRepository) UpdateLocal(
	ctx context.Context,
	value artifactModel.Artifact,
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
	ref artifactModel.ArtifactRef,
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
	rootID rootModel.RootID,
	digest cryptoutil.Digest,
) (definitionModel.Definition, error) {
	return r.store.getDefinition(ctx, rootID, digest)
}

func (r *DefinitionRepository) GetDefinitions(
	ctx context.Context,
	keys []definitionModel.Key,
) ([]definitionModel.Definition, error) {
	return r.store.getDefinitions(ctx, keys)
}

func (r *RefreshStateRepository) GetRefreshState(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (sourceModel.RefreshState, error) {
	return r.store.getRefreshState(ctx, rootID, sourceID)
}

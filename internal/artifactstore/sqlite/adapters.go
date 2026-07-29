package sqlite

import (
	"context"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type SourceRepository struct {
	store *Store
}

type RootRepository struct {
	store *Store
}

type CollectionRepository struct {
	store *Store
}

type CatalogRepository struct {
	store *Store
}

type ArtifactRepository struct {
	store *Store
}

func (s *Store) Sources() *SourceRepository {
	return &SourceRepository{store: s}
}

func (s *Store) Roots() *RootRepository {
	return &RootRepository{store: s}
}

func (s *Store) Collections() *CollectionRepository {
	return &CollectionRepository{store: s}
}

func (s *Store) Artifacts() *ArtifactRepository {
	return &ArtifactRepository{store: s}
}

func (s *Store) Catalogs() *CatalogRepository {
	return &CatalogRepository{store: s}
}

func (r *SourceRepository) Create(
	ctx context.Context,
	value source.Source,
) error {
	return r.store.createSource(ctx, value)
}

func (r *SourceRepository) Get(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
) (source.Source, error) {
	return r.store.getSource(ctx, rootID, id)
}

func (r *SourceRepository) List(
	ctx context.Context,
	rootID basespec.RootID,
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
	rootID basespec.RootID,
	id basespec.SourceID,
	expectedRevision uint64,
) error {
	return r.store.discardSource(ctx, rootID, id, expectedRevision)
}

func (r *SourceRepository) Purge(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
	expectedRevision uint64,
) error {
	return r.store.purgeSource(ctx, rootID, id, expectedRevision)
}

func (r *RootRepository) Create(
	ctx context.Context,
	value root.Root,
) error {
	return r.store.createRoot(ctx, value)
}

func (r *RootRepository) Get(
	ctx context.Context,
	id basespec.RootID,
) (root.Root, error) {
	return r.store.getRoot(ctx, id)
}

func (r *RootRepository) List(ctx context.Context) ([]root.Root, error) {
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
	id basespec.RootID,
	expectedRevision uint64,
) error {
	return r.store.purgeRoot(ctx, id, expectedRevision)
}

func (r *CollectionRepository) Create(
	ctx context.Context,
	value collection.Collection,
	attachments []collection.Attachment,
) error {
	return r.store.createCollection(ctx, value, attachments)
}

func (r *CollectionRepository) Get(
	ctx context.Context,
	ref basespec.CollectionRef,
) (collection.Collection, error) {
	return r.store.getCollection(ctx, ref)
}

func (r *CollectionRepository) GetRetired(
	ctx context.Context,
	ref basespec.CollectionRef,
) (collection.Collection, error) {
	return r.store.getRetiredCollection(ctx, ref)
}

func (r *CollectionRepository) ListByRoot(
	ctx context.Context,
	rootID basespec.RootID,
) ([]collection.Collection, error) {
	return r.store.listCollectionsByRoot(ctx, rootID)
}

func (r *CollectionRepository) Update(
	ctx context.Context,
	value collection.Collection,
	expectedRevision uint64,
) error {
	return r.store.updateCollection(ctx, value, expectedRevision)
}

func (r *CollectionRepository) Retire(
	ctx context.Context,
	value collection.Collection,
	expectedRevision uint64,
) error {
	return r.store.retireCollection(ctx, value, expectedRevision)
}

func (r *CollectionRepository) Purge(
	ctx context.Context,
	ref basespec.CollectionRef,
	expectedRevision uint64,
) error {
	return r.store.purgeCollection(ctx, ref, expectedRevision)
}

func (r *CollectionRepository) Attach(
	ctx context.Context,
	value collection.Attachment,
	expectedCollectionRevision uint64,
) (collection.Collection, error) {
	return r.store.attachCollectionSource(
		ctx,
		value,
		expectedCollectionRevision,
	)
}

func (r *CollectionRepository) GetAttachment(
	ctx context.Context,
	ref basespec.CollectionRef,
	sourceID basespec.SourceID,
) (collection.Attachment, error) {
	return r.store.getCollectionAttachment(ctx, ref, sourceID)
}

func (r *CollectionRepository) ListAttachments(
	ctx context.Context,
	ref basespec.CollectionRef,
) ([]collection.Attachment, error) {
	return r.store.listCollectionAttachments(ctx, ref)
}

func (r *CollectionRepository) UpdateAttachment(
	ctx context.Context,
	value collection.Attachment,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
) (collection.Collection, error) {
	return r.store.updateCollectionAttachment(
		ctx,
		value,
		expectedCollectionRevision,
		expectedAttachmentRevision,
	)
}

func (r *CollectionRepository) Detach(
	ctx context.Context,
	ref basespec.CollectionRef,
	sourceID basespec.SourceID,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
	modifiedAt time.Time,
) (collection.Collection, error) {
	return r.store.detachCollectionSource(
		ctx,
		ref,
		sourceID,
		expectedCollectionRevision,
		expectedAttachmentRevision,
		modifiedAt,
	)
}

func (r *CollectionRepository) ReplaceAttachment(
	ctx context.Context,
	ref basespec.CollectionRef,
	previousSourceID basespec.SourceID,
	expectedPreviousRevision uint64,
	replacement collection.Attachment,
	expectedCollectionRevision uint64,
) (collection.Collection, error) {
	return r.store.replaceCollectionAttachment(
		ctx,
		ref,
		previousSourceID,
		expectedPreviousRevision,
		replacement,
		expectedCollectionRevision,
	)
}

func (r *CatalogRepository) GetCurrent(
	ctx context.Context,
	ref basespec.CollectionRef,
) (catalog.Snapshot, error) {
	return r.store.getCurrentCatalog(ctx, ref)
}

func (r *ArtifactRepository) Get(
	ctx context.Context,
	ref basespec.ArtifactRef,
) (artifact.Artifact, error) {
	return r.store.getArtifact(ctx, ref)
}

func (r *ArtifactRepository) ListByCollection(
	ctx context.Context,
	ref basespec.CollectionRef,
) ([]artifact.Artifact, error) {
	return r.store.listArtifactsByCollection(ctx, ref)
}

func (r *ArtifactRepository) ListSuppressions(
	ctx context.Context,
	ref basespec.CollectionRef,
) ([]artifact.Suppression, error) {
	return r.store.listSuppressions(ctx, ref)
}

func (r *ArtifactRepository) Update(
	ctx context.Context,
	value artifact.Artifact,
	expectedRevision uint64,
) error {
	return r.store.updateArtifact(ctx, value, expectedRevision)
}

func (r *ArtifactRepository) CreateAdopted(
	ctx context.Context,
	value artifact.Artifact,
	expectedCollectionRevision uint64,
	expectedCatalogRevision uint64,
) error {
	return r.store.createAdoptedArtifact(
		ctx,
		value,
		expectedCollectionRevision,
		expectedCatalogRevision,
	)
}

func (r *ArtifactRepository) CreatePinned(
	ctx context.Context,
	value artifact.Artifact,
	expectedCollectionRevision uint64,
	expectedCatalogRevision uint64,
) error {
	return r.store.createPinnedArtifact(
		ctx,
		value,
		expectedCollectionRevision,
		expectedCatalogRevision,
	)
}

func (r *ArtifactRepository) Unadopt(
	ctx context.Context,
	ref basespec.ArtifactRef,
	expectedRevision uint64,
	suppression *artifact.Suppression,
) error {
	return r.store.unadoptArtifact(
		ctx,
		ref,
		expectedRevision,
		suppression,
	)
}

func (r *ArtifactRepository) Suppress(
	ctx context.Context,
	value artifact.Suppression,
	expectedCollectionRevision uint64,
) error {
	return r.store.createSuppression(
		ctx,
		value,
		expectedCollectionRevision,
	)
}

func (r *ArtifactRepository) Unsuppress(
	ctx context.Context,
	ref basespec.CollectionRef,
	binding basespec.SourceBinding,
	expectedRevision uint64,
) error {
	return r.store.deleteSuppression(
		ctx,
		ref,
		binding,
		expectedRevision,
	)
}

func (r *ArtifactRepository) Purge(
	ctx context.Context,
	ref basespec.ArtifactRef,
	expectedRevision uint64,
) error {
	return r.store.purgeArtifact(
		ctx,
		ref,
		expectedRevision,
	)
}

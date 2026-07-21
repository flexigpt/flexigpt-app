package sqlite

import (
	"context"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type SourceRepository struct {
	store *Store
}

type CatalogRepository struct {
	store *Store
}

type RecordRepository struct {
	store *Store
}

func (s *Store) Sources() *SourceRepository {
	return &SourceRepository{store: s}
}

func (s *Store) Catalog() *CatalogRepository {
	return &CatalogRepository{store: s}
}

func (s *Store) Records() *RecordRepository {
	return &RecordRepository{store: s}
}

func (r *SourceRepository) Create(
	ctx context.Context,
	value source.Source,
) error {
	return r.store.createSource(ctx, value)
}

func (r *SourceRepository) Get(
	ctx context.Context,
	id artifactstore.SourceID,
) (source.Source, error) {
	return r.store.getSource(ctx, id)
}

func (r *SourceRepository) List(
	ctx context.Context,
) ([]source.Source, error) {
	return r.store.listSources(ctx)
}

func (r *SourceRepository) Update(
	ctx context.Context,
	value source.Source,
	expectedRevision uint64,
) error {
	return r.store.updateSource(ctx, value, expectedRevision)
}

func (r *SourceRepository) Delete(
	ctx context.Context,
	id artifactstore.SourceID,
	expectedRevision uint64,
) error {
	return r.store.deleteSource(ctx, id, expectedRevision)
}

func (r *CatalogRepository) CreateRoot(
	ctx context.Context,
	root catalog.Root,
	attachments []catalog.Attachment,
) error {
	return r.store.createRoot(ctx, root, attachments)
}

func (r *CatalogRepository) GetRoot(
	ctx context.Context,
	id artifactstore.RootID,
	includeDeleted bool,
) (catalog.Root, error) {
	return r.store.getRoot(ctx, id, includeDeleted)
}

func (r *CatalogRepository) ListRoots(
	ctx context.Context,
	includeDeleted bool,
) ([]catalog.Root, error) {
	return r.store.listRoots(ctx, includeDeleted)
}

func (r *CatalogRepository) UpdateRoot(
	ctx context.Context,
	value catalog.Root,
	expectedRevision uint64,
) error {
	return r.store.updateRoot(ctx, value, expectedRevision)
}

func (r *CatalogRepository) Attach(
	ctx context.Context,
	value catalog.Attachment,
	expectedRootRevision uint64,
) (catalog.Root, error) {
	return r.store.attach(ctx, value, expectedRootRevision)
}

func (r *CatalogRepository) GetAttachment(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
) (catalog.Attachment, error) {
	return r.store.getAttachment(ctx, rootID, sourceID)
}

func (r *CatalogRepository) ListAttachments(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]catalog.Attachment, error) {
	return r.store.listAttachments(ctx, rootID)
}

func (r *CatalogRepository) UpdateAttachment(
	ctx context.Context,
	value catalog.Attachment,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
) (catalog.Root, error) {
	return r.store.updateAttachment(
		ctx,
		value,
		expectedRootRevision,
		expectedAttachmentRevision,
	)
}

func (r *CatalogRepository) Detach(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
	modifiedAt time.Time,
) (catalog.Root, error) {
	return r.store.detach(
		ctx,
		rootID,
		sourceID,
		expectedRootRevision,
		expectedAttachmentRevision,
		modifiedAt,
	)
}

func (r *CatalogRepository) GetCurrentCatalog(
	ctx context.Context,
	rootID artifactstore.RootID,
) (catalog.Snapshot, error) {
	return r.store.getCurrentCatalog(ctx, rootID)
}

func (r *RecordRepository) Get(
	ctx context.Context,
	id artifactstore.RecordID,
) (record.Record, error) {
	return r.store.getRecord(ctx, id)
}

func (r *RecordRepository) ListByRoot(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]record.Record, error) {
	return r.store.listRecordsByRoot(ctx, rootID)
}

func (r *RecordRepository) Update(
	ctx context.Context,
	value record.Record,
	expectedRevision uint64,
) error {
	return r.store.updateRecord(ctx, value, expectedRevision)
}

func (r *RecordRepository) Delete(
	ctx context.Context,
	id artifactstore.RecordID,
	expectedRevision uint64,
) error {
	return r.store.deleteRecord(ctx, id, expectedRevision)
}

var (
	_ source.Repository  = (*SourceRepository)(nil)
	_ catalog.Repository = (*CatalogRepository)(nil)
	_ record.Repository  = (*RecordRepository)(nil)
)

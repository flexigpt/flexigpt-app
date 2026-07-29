package engine

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

var errEngineTestUnexpected = errors.New("unexpected engine test dependency call")

type engineTestCollectionStore struct {
	createFn            func(context.Context, basespec.RootID, collection.Draft, []collection.AttachmentDraft) (collection.Collection, []collection.Attachment, error)
	getFn               func(context.Context, collection.CollectionRef) (collection.Collection, error)
	getRetiredFn        func(context.Context, collection.CollectionRef) (collection.Collection, error)
	listByRootFn        func(context.Context, basespec.RootID) ([]collection.Collection, error)
	updateFn            func(context.Context, collection.CollectionRef, collection.Update) (collection.Collection, error)
	retireFn            func(context.Context, collection.CollectionRef, uint64) (collection.Collection, error)
	purgeFn             func(context.Context, collection.CollectionRef, uint64) error
	attachFn            func(context.Context, collection.CollectionRef, uint64, collection.AttachmentDraft) (collection.Collection, collection.Attachment, error)
	getAttachmentFn     func(context.Context, collection.CollectionRef, basespec.SourceID) (collection.Attachment, error)
	listAttachmentsFn   func(context.Context, collection.CollectionRef) ([]collection.Attachment, error)
	updateAttachmentFn  func(context.Context, collection.CollectionRef, basespec.SourceID, collection.AttachmentUpdate) (collection.Collection, collection.Attachment, error)
	detachFn            func(context.Context, collection.CollectionRef, basespec.SourceID, uint64, uint64) (collection.Collection, error)
	replaceAttachmentFn func(context.Context, collection.CollectionRef, collection.AttachmentReplacement) (collection.Collection, collection.Attachment, error)
}

func (s *engineTestCollectionStore) Create(
	ctx context.Context,
	rootID basespec.RootID,
	draft collection.Draft,
	attachments []collection.AttachmentDraft,
) (collection.Collection, []collection.Attachment, error) {
	if s.createFn == nil {
		return collection.Collection{}, nil, errEngineTestUnexpected
	}
	return s.createFn(ctx, rootID, draft, attachments)
}

func (s *engineTestCollectionStore) Get(
	ctx context.Context,
	ref collection.CollectionRef,
) (collection.Collection, error) {
	if s.getFn == nil {
		return collection.Collection{}, errEngineTestUnexpected
	}
	return s.getFn(ctx, ref)
}

func (s *engineTestCollectionStore) GetRetired(
	ctx context.Context,
	ref collection.CollectionRef,
) (collection.Collection, error) {
	if s.getRetiredFn == nil {
		return collection.Collection{}, errEngineTestUnexpected
	}
	return s.getRetiredFn(ctx, ref)
}

func (s *engineTestCollectionStore) ListByRoot(
	ctx context.Context,
	rootID basespec.RootID,
) ([]collection.Collection, error) {
	if s.listByRootFn == nil {
		return nil, errEngineTestUnexpected
	}
	return s.listByRootFn(ctx, rootID)
}

func (s *engineTestCollectionStore) Update(
	ctx context.Context,
	ref collection.CollectionRef,
	update collection.Update,
) (collection.Collection, error) {
	if s.updateFn == nil {
		return collection.Collection{}, errEngineTestUnexpected
	}
	return s.updateFn(ctx, ref, update)
}

func (s *engineTestCollectionStore) Retire(
	ctx context.Context,
	ref collection.CollectionRef,
	revision uint64,
) (collection.Collection, error) {
	if s.retireFn == nil {
		return collection.Collection{}, errEngineTestUnexpected
	}
	return s.retireFn(ctx, ref, revision)
}

func (s *engineTestCollectionStore) Purge(ctx context.Context, ref collection.CollectionRef, revision uint64) error {
	if s.purgeFn == nil {
		return errEngineTestUnexpected
	}
	return s.purgeFn(ctx, ref, revision)
}

func (s *engineTestCollectionStore) Attach(
	ctx context.Context,
	ref collection.CollectionRef,
	revision uint64,
	draft collection.AttachmentDraft,
) (collection.Collection, collection.Attachment, error) {
	if s.attachFn == nil {
		return collection.Collection{}, collection.Attachment{}, errEngineTestUnexpected
	}
	return s.attachFn(ctx, ref, revision, draft)
}

func (s *engineTestCollectionStore) GetAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
) (collection.Attachment, error) {
	if s.getAttachmentFn == nil {
		return collection.Attachment{}, errEngineTestUnexpected
	}
	return s.getAttachmentFn(ctx, ref, sourceID)
}

func (s *engineTestCollectionStore) ListAttachments(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]collection.Attachment, error) {
	if s.listAttachmentsFn == nil {
		return nil, errEngineTestUnexpected
	}
	return s.listAttachmentsFn(ctx, ref)
}

func (s *engineTestCollectionStore) UpdateAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	update collection.AttachmentUpdate,
) (collection.Collection, collection.Attachment, error) {
	if s.updateAttachmentFn == nil {
		return collection.Collection{}, collection.Attachment{}, errEngineTestUnexpected
	}
	return s.updateAttachmentFn(ctx, ref, sourceID, update)
}

func (s *engineTestCollectionStore) Detach(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	collectionRevision, attachmentRevision uint64,
) (collection.Collection, error) {
	if s.detachFn == nil {
		return collection.Collection{}, errEngineTestUnexpected
	}
	return s.detachFn(ctx, ref, sourceID, collectionRevision, attachmentRevision)
}

func (s *engineTestCollectionStore) ReplaceAttachment(
	ctx context.Context,
	ref collection.CollectionRef,
	replacement collection.AttachmentReplacement,
) (collection.Collection, collection.Attachment, error) {
	if s.replaceAttachmentFn == nil {
		return collection.Collection{}, collection.Attachment{}, errEngineTestUnexpected
	}
	return s.replaceAttachmentFn(ctx, ref, replacement)
}

type engineTestSources struct {
	getFn func(context.Context, basespec.RootID, basespec.SourceID) (source.Summary, error)
}

func (s engineTestSources) Get(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
) (source.Summary, error) {
	if s.getFn == nil {
		return source.Summary{}, errEngineTestUnexpected
	}
	return s.getFn(ctx, rootID, id)
}

type engineTestCatalogs struct {
	getFn func(context.Context, collection.CollectionRef) (catalog.Snapshot, error)
}

func (c engineTestCatalogs) GetCurrent(ctx context.Context, ref collection.CollectionRef) (catalog.Snapshot, error) {
	if c.getFn == nil {
		return catalog.Snapshot{}, errEngineTestUnexpected
	}
	return c.getFn(ctx, ref)
}

type engineTestArtifacts struct {
	getFn  func(context.Context, artifact.ArtifactRef) (artifact.Artifact, error)
	listFn func(context.Context, collection.CollectionRef) ([]artifact.Artifact, error)
}

func (a engineTestArtifacts) Get(ctx context.Context, ref artifact.ArtifactRef) (artifact.Artifact, error) {
	if a.getFn == nil {
		return artifact.Artifact{}, errEngineTestUnexpected
	}
	return a.getFn(ctx, ref)
}

func (a engineTestArtifacts) ListByCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	if a.listFn == nil {
		return nil, errEngineTestUnexpected
	}
	return a.listFn(ctx, ref)
}

type engineTestDefinitions struct {
	getFn func(context.Context, basespec.RootID, cryptoutil.Digest) (definition.Definition, error)
}

func (d engineTestDefinitions) Get(
	ctx context.Context,
	rootID basespec.RootID,
	digest cryptoutil.Digest,
) (definition.Definition, error) {
	if d.getFn == nil {
		return definition.Definition{}, errEngineTestUnexpected
	}
	return d.getFn(ctx, rootID, digest)
}

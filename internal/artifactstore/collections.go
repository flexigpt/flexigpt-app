package artifactstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

// CollectionUpdate replaces mutable local collection fields. CollectionID,
// RootID, Kind, Slug, timestamps, and soft-deletion state remain store-owned.
type CollectionUpdate struct {
	ExpectedModifiedAt time.Time
	DisplayName        string
	Description        string
	Enabled            bool
	DataSchemaID       spec.SchemaID
	Data               json.RawMessage
}

// EnsureBaseCollection returns an active collection with the requested root and
// slug, creating it when absent. It handles a concurrent creator by reloading
// the natural key after a uniqueness conflict.
func (s *Store) EnsureBaseCollection(ctx context.Context, draft spec.CollectionDraft) (spec.ArtifactCollection, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	defer finish()
	collection, err := s.repository.GetCollectionByRootSlug(ctx, draft.RootID, draft.Slug, true)
	if err == nil {
		if collection.SoftDeletedAt != nil {
			return spec.ArtifactCollection{}, fmt.Errorf(
				"%w: collection %q is soft-deleted",
				spec.ErrConflict,
				collection.CollectionID,
			)
		}
		if collection.RootID != draft.RootID || collection.Kind != draft.Kind {
			return spec.ArtifactCollection{}, fmt.Errorf(
				"%w: collection slug %q already belongs to kind %q",
				spec.ErrConflict,
				draft.Slug,
				collection.Kind,
			)
		}
		return collection, nil
	}
	if !isNotFound(err) {
		return spec.ArtifactCollection{}, err
	}
	collection, err = s.CreateCollection(ctx, draft)
	if err == nil {
		return collection, nil
	}
	if !isConflict(err) {
		return spec.ArtifactCollection{}, err
	}
	collection, reloadErr := s.repository.GetCollectionByRootSlug(ctx, draft.RootID, draft.Slug, true)
	if reloadErr != nil {
		return spec.ArtifactCollection{}, errors.Join(err, reloadErr)
	}
	if collection.SoftDeletedAt != nil {
		return spec.ArtifactCollection{}, fmt.Errorf(
			"%w: collection %q is soft-deleted",
			spec.ErrConflict,
			collection.CollectionID,
		)
	}
	if collection.RootID != draft.RootID || collection.Kind != draft.Kind {
		return spec.ArtifactCollection{}, fmt.Errorf(
			"%w: collection slug %q already belongs to kind %q",
			spec.ErrConflict,
			draft.Slug,
			collection.Kind,
		)
	}
	return collection, nil
}

// CreateCollection creates an app-local grouping of records.
func (s *Store) CreateCollection(ctx context.Context, draft spec.CollectionDraft) (spec.ArtifactCollection, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	defer finish()
	if _, err := s.repository.GetRoot(ctx, draft.RootID, false); err != nil {
		return spec.ArtifactCollection{}, err
	}
	id, err := s.newID()
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	now := s.nowUTC()
	collection := spec.ArtifactCollection{
		CollectionID: spec.CollectionID(id),
		RootID:       draft.RootID,
		Kind:         draft.Kind,
		Slug:         draft.Slug,
		DisplayName:  draft.DisplayName,
		Description:  draft.Description,
		Enabled:      draft.Enabled,
		DataSchemaID: draft.DataSchemaID,
		Data:         normalizedJSONObject(draft.Data),
		CreatedAt:    now,
		ModifiedAt:   now,
	}
	if err := s.validateCollection(ctx, collection); err != nil {
		return spec.ArtifactCollection{}, err
	}
	if err := s.repository.CreateCollection(ctx, collection); err != nil {
		return spec.ArtifactCollection{}, err
	}
	return collection, nil
}

// GetCollection returns one active collection.
func (s *Store) GetCollection(ctx context.Context, collectionID spec.CollectionID) (spec.ArtifactCollection, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	defer finish()
	return s.repository.GetCollection(ctx, collectionID, false)
}

// GetCollectionIncludingDeleted returns a collection regardless of soft deletion.
func (s *Store) GetCollectionIncludingDeleted(
	ctx context.Context,
	collectionID spec.CollectionID,
) (spec.ArtifactCollection, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	defer finish()
	return s.repository.GetCollection(ctx, collectionID, true)
}

// ListCollections lists collections belonging to one root.
func (s *Store) ListCollections(
	ctx context.Context,
	rootID spec.RootID,
	includeSoftDeleted bool,
) ([]spec.ArtifactCollection, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	if _, err := s.repository.GetRoot(ctx, rootID, false); err != nil {
		return nil, err
	}
	return s.repository.ListCollections(ctx, rootID, includeSoftDeleted)
}

// UpdateCollection replaces mutable local collection fields.
func (s *Store) UpdateCollection(
	ctx context.Context,
	collectionID spec.CollectionID,
	update CollectionUpdate,
) (spec.ArtifactCollection, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	defer finish()

	collection, err := s.repository.GetCollection(ctx, collectionID, true)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	if err := requireExpectedModifiedAt(
		"collection "+string(collectionID),
		collection.ModifiedAt,
		update.ExpectedModifiedAt,
	); err != nil {
		return spec.ArtifactCollection{}, err
	}
	if collection.SoftDeletedAt != nil {
		return spec.ArtifactCollection{}, fmt.Errorf(
			"%w: collection %q is soft-deleted",
			spec.ErrConflict,
			collectionID,
		)
	}
	if _, err := s.repository.GetRoot(ctx, collection.RootID, false); err != nil {
		return spec.ArtifactCollection{}, err
	}
	collection.DisplayName = update.DisplayName
	collection.Description = update.Description
	collection.Enabled = update.Enabled
	collection.DataSchemaID = update.DataSchemaID
	collection.Data = normalizedJSONObject(update.Data)
	collection.ModifiedAt = s.nextModifiedAt(collection.ModifiedAt)
	if err := s.validateCollection(ctx, collection); err != nil {
		return spec.ArtifactCollection{}, err
	}
	if err := s.repository.UpdateCollection(ctx, collection, update.ExpectedModifiedAt); err != nil {
		return spec.ArtifactCollection{}, err
	}
	return collection, nil
}

// DeleteCollection disables and soft-deletes an empty collection.
func (s *Store) DeleteCollection(
	ctx context.Context,
	collectionID spec.CollectionID,
	expectedModifiedAt time.Time,
) (spec.ArtifactCollection, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	defer finish()

	collection, err := s.repository.GetCollection(ctx, collectionID, true)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	if err := requireExpectedModifiedAt(
		"collection "+string(collectionID),
		collection.ModifiedAt,
		expectedModifiedAt,
	); err != nil {
		return spec.ArtifactCollection{}, err
	}
	if collection.SoftDeletedAt != nil {
		return spec.ArtifactCollection{}, fmt.Errorf(
			"%w: collection %q is already soft-deleted",
			spec.ErrConflict,
			collectionID,
		)
	}
	count, err := s.repository.CountRecordsInCollection(ctx, collectionID)
	if err != nil {
		return spec.ArtifactCollection{}, err
	}
	if count != 0 {
		return spec.ArtifactCollection{}, fmt.Errorf(
			"%w: collection %q still contains %d record(s)",
			spec.ErrConflict,
			collectionID,
			count,
		)
	}
	now := s.nextModifiedAt(collection.ModifiedAt)
	collection.Enabled = false
	collection.ModifiedAt = now
	collection.SoftDeletedAt = &now
	if err := s.validateCollection(ctx, collection); err != nil {
		return spec.ArtifactCollection{}, err
	}
	if err := s.repository.UpdateCollection(ctx, collection, expectedModifiedAt); err != nil {
		return spec.ArtifactCollection{}, err
	}
	return collection, nil
}

func (s *Store) validateCollection(ctx context.Context, collection spec.ArtifactCollection) error {
	if err := validate.ValidateArtifactCollection(collection); err != nil {
		return fmt.Errorf("%w: collection: %w", spec.ErrInvalidRequest, err)
	}
	if hook, ok := s.collectionHookFor(collection.Kind); ok {
		if err := errorDiagnostics(
			"collection "+string(collection.Kind),
			hook.ValidateCollectionData(ctx, collection),
		); err != nil {
			return err
		}
	}
	return nil
}

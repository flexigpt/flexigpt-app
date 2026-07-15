package artifactstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

// RootUpdate is a full mutable replacement for an active root's local fields.
// RootID, Kind, CreatedAt, and soft-deletion state remain store-owned.
type RootUpdate struct {
	ExpectedModifiedAt time.Time
	DisplayName        string
	Description        string
	Enabled            bool
	DataSchemaID       spec.SchemaID
	Data               json.RawMessage
}

// CreateRoot creates app-local root metadata after generic and typed root
// validation. It does not create or modify any portable source file.
func (s *Store) CreateRoot(ctx context.Context, draft spec.RootDraft) (spec.ArtifactRoot, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	defer finish()
	if err := ctx.Err(); err != nil {
		return spec.ArtifactRoot{}, err
	}
	id, err := s.newID()
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	now := s.nowUTC()
	root := spec.ArtifactRoot{
		RootID:       spec.RootID(id),
		Kind:         draft.Kind,
		DisplayName:  draft.DisplayName,
		Description:  draft.Description,
		Enabled:      draft.Enabled,
		DataSchemaID: draft.DataSchemaID,
		Data:         normalizedJSONObject(draft.Data),
		CreatedAt:    now,
		ModifiedAt:   now,
	}
	if err := s.validateRoot(ctx, root); err != nil {
		return spec.ArtifactRoot{}, err
	}
	if err := s.repository.CreateRoot(ctx, root); err != nil {
		return spec.ArtifactRoot{}, err
	}
	return root, nil
}

// GetRoot returns an active root. Soft-deleted roots are intentionally hidden.
func (s *Store) GetRoot(ctx context.Context, rootID spec.RootID) (spec.ArtifactRoot, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	defer finish()
	return s.repository.GetRoot(ctx, rootID, false)
}

// GetRootIncludingDeleted returns a root regardless of its soft-deletion state.
func (s *Store) GetRootIncludingDeleted(ctx context.Context, rootID spec.RootID) (spec.ArtifactRoot, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	defer finish()
	ctx, finish, err = s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	defer finish()
	return s.repository.GetRoot(ctx, rootID, true)
}

// ListRoots lists roots in descending modification order.
func (s *Store) ListRoots(ctx context.Context, includeSoftDeleted bool) ([]spec.ArtifactRoot, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	return s.repository.ListRoots(ctx, includeSoftDeleted)
}

// UpdateRoot replaces mutable app-local root fields.
func (s *Store) UpdateRoot(ctx context.Context, rootID spec.RootID, update RootUpdate) (spec.ArtifactRoot, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	defer finish()

	current, err := s.repository.GetRoot(ctx, rootID, true)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	if err := requireExpectedModifiedAt(
		"root "+string(rootID),
		current.ModifiedAt,
		update.ExpectedModifiedAt,
	); err != nil {
		return spec.ArtifactRoot{}, err
	}
	if current.SoftDeletedAt != nil {
		return spec.ArtifactRoot{}, fmt.Errorf("%w: root %q is soft-deleted", spec.ErrConflict, rootID)
	}
	current.DisplayName = update.DisplayName
	current.Description = update.Description
	current.Enabled = update.Enabled
	current.DataSchemaID = update.DataSchemaID
	current.Data = normalizedJSONObject(update.Data)
	current.ModifiedAt = s.nextModifiedAt(current.ModifiedAt)
	if err := s.validateRoot(ctx, current); err != nil {
		return spec.ArtifactRoot{}, err
	}
	if err := s.repository.UpdateRoot(ctx, current, update.ExpectedModifiedAt); err != nil {
		return spec.ArtifactRoot{}, err
	}
	return current, nil
}

// DeleteRoot marks a root as disabled and soft-deleted. Associated local data
// remains intact and can be inspected through explicit including-deleted APIs.
func (s *Store) DeleteRoot(
	ctx context.Context,
	rootID spec.RootID,
	expectedModifiedAt time.Time,
) (spec.ArtifactRoot, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	defer finish()

	current, err := s.repository.GetRoot(ctx, rootID, true)
	if err != nil {
		return spec.ArtifactRoot{}, err
	}
	if err := requireExpectedModifiedAt("root "+string(rootID), current.ModifiedAt, expectedModifiedAt); err != nil {
		return spec.ArtifactRoot{}, err
	}
	if current.SoftDeletedAt != nil {
		return spec.ArtifactRoot{}, fmt.Errorf("%w: root %q is already soft-deleted", spec.ErrConflict, rootID)
	}
	now := s.nextModifiedAt(current.ModifiedAt)
	current.Enabled = false
	current.ModifiedAt = now
	current.SoftDeletedAt = &now
	if err := s.validateRoot(ctx, current); err != nil {
		return spec.ArtifactRoot{}, err
	}
	if err := s.repository.UpdateRoot(ctx, current, expectedModifiedAt); err != nil {
		return spec.ArtifactRoot{}, err
	}
	return current, nil
}

func (s *Store) validateRoot(ctx context.Context, root spec.ArtifactRoot) error {
	if err := validate.ValidateArtifactRoot(root); err != nil {
		return fmt.Errorf("%w: root: %w", spec.ErrInvalidRequest, err)
	}
	if hook, ok := s.rootHookFor(root.Kind); ok {
		if err := errorDiagnostics("root "+string(root.Kind), hook.ValidateRootData(ctx, root)); err != nil {
			return err
		}
	}
	return nil
}

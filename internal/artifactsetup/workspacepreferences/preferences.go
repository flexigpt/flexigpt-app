// Package workspacepreferences persists the optional application default
// Workspace selection. Workspace-family resolution remains in
// llmartifactory-go/workspace.
package workspacepreferences

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const schemaVersion = "v1"

const preferencesNamespace overlayModel.Namespace = "workspace.preferences"

// StoreNamespaces returns the Store-scoped namespaces owned by this package.
func StoreNamespaces() []overlayModel.Namespace {
	return []overlayModel.Namespace{preferencesNamespace}
}

type payload struct {
	Workspace *artifactModel.ArtifactRef `json:"workspace,omitempty"`
}

// Store owns application preference persistence. Workspace-family effective
// Workspace validation and runtime resolution remain in llmartifactory-go.
type Store struct {
	overlays overlay.StoreAPI
}

func New(
	overlays overlay.StoreAPI,
) (*Store, error) {
	if overlays == nil {
		return nil, fmt.Errorf(
			"%w: Workspace preference overlay store is required",
			storeSpec.ErrInvalid,
		)
	}
	return &Store{overlays: overlays}, nil
}

// GetDefaultWorkspace returns nil when no default Workspace is configured.
func (s *Store) GetDefaultWorkspace(
	ctx context.Context,
) (*artifactModel.ArtifactRef, error) {
	record, found, err := s.overlays.GetStoreOverlay(
		ctx,
		preferencesNamespace,
	)
	if err != nil || !found {
		return nil, err
	}

	var value payload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&value,
		storeSpec.MaxLocalDataBytes,
	); err != nil {
		return nil, fmt.Errorf(
			"%w: decode Workspace default preference: %w",
			storeSpec.ErrInvalid,
			err,
		)
	}
	if value.Workspace == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}
	if err := value.Workspace.Validate(); err != nil {
		return nil, err
	}

	output := *value.Workspace
	return &output, nil
}

// SetDefaultWorkspace saves the optional default Workspace. A nil value clears
// the preference and is the supported no-default state.
func (s *Store) SetDefaultWorkspace(
	ctx context.Context,
	workspace *artifactModel.ArtifactRef,
) error {
	if workspace != nil {
		if err := workspace.Validate(); err != nil {
			return err
		}
	}

	current, found, err := s.overlays.GetStoreOverlay(
		ctx,
		preferencesNamespace,
	)
	if err != nil {
		return err
	}

	if workspace == nil {
		if !found {
			return nil
		}
		return s.overlays.DeleteStoreOverlay(
			ctx,
			preferencesNamespace,
			current.Revision,
		)
	}

	value := payload{Workspace: workspace}
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		storeSpec.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}

	expectedRevision := uint64(0)
	if found {
		expectedRevision = current.Revision
	}

	_, err = s.overlays.PutStoreOverlay(
		ctx,
		overlayModel.StorePutRequest{
			Namespace:        preferencesNamespace,
			SchemaVersion:    schemaVersion,
			Payload:          raw,
			ExpectedRevision: expectedRevision,
		},
	)
	return err
}

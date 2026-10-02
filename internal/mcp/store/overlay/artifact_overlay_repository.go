package overlay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	artifactOverlay "github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/overlay"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type ArtifactOverlayRepository struct {
	artifacts        compositionapi.ArtifactAPI
	protection       compositionapi.ProtectionAPI
	protectedOverlay compositionapi.ProtectedOverlayAPI
	localState       compositionapi.LocalStateMaintenanceAPI
}

func NewArtifactOverlayRepository(
	dependencies ArtifactOverlayDependencies,
) (*ArtifactOverlayRepository, error) {
	if dependencies.Artifacts == nil ||
		dependencies.Protection == nil ||
		dependencies.ProtectedOverlay == nil ||
		dependencies.LocalState == nil {
		return nil, fmt.Errorf(
			"%w: MCP Artifact overlay dependencies are incomplete",
			model.ErrInvalid,
		)
	}

	return &ArtifactOverlayRepository{
		artifacts:        dependencies.Artifacts,
		protection:       dependencies.Protection,
		protectedOverlay: dependencies.ProtectedOverlay,
		localState:       dependencies.LocalState,
	}, nil
}

func (r *ArtifactOverlayRepository) GetServerOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerOverlay, bool, error) {
	record, err := r.artifact(ctx, ref)
	if err != nil {
		return ServerOverlay{}, false, err
	}
	if !r.protection.IsProtectedRoot(record.RootID) {
		return ServerOverlay{}, false, nil
	}

	stored, found, err := r.protectedOverlay.Get(
		ctx,
		ref,
		InstallationNamespace,
	)
	if err != nil || !found {
		return ServerOverlay{}, found, err
	}

	value, err := decodeProtectedServerOverlay(stored)
	if err != nil {
		return ServerOverlay{}, false, err
	}
	return value.Clone(), true, nil
}

func (r *ArtifactOverlayRepository) PutServerOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
	value ServerOverlay,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := validateOverlayTransition(
		expectedOverlayRevision,
		value.Revision,
	); err != nil {
		return err
	}

	record, err := r.artifact(ctx, ref)
	if err != nil {
		return err
	}
	if !r.protection.IsProtectedRoot(record.RootID) {
		return fmt.Errorf(
			"%w: MCP server is not in a protected Root",
			model.ErrProtected,
		)
	}
	if record.Revision != expectedArtifactRevision {
		return model.ErrConflict
	}

	payload, err := encodeProtectedServerOverlay(value)
	if err != nil {
		return err
	}

	stored, err := r.protectedOverlay.Put(
		ctx,
		artifactOverlay.PutRequest{
			Artifact:                 ref,
			Namespace:                InstallationNamespace,
			SchemaVersion:            value.SchemaVersion,
			Payload:                  payload,
			ExpectedArtifactRevision: expectedArtifactRevision,
			ExpectedOverlayRevision:  expectedOverlayRevision,
		},
	)
	if err != nil {
		return err
	}
	if stored.Revision != value.Revision {
		return fmt.Errorf(
			"%w: MCP protected overlay revision changed unexpectedly",
			model.ErrConflict,
		)
	}
	return nil
}

func (r *ArtifactOverlayRepository) DeleteServerOverlay(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	expectedOverlayRevision uint64,
) error {
	record, err := r.artifact(ctx, ref)
	if err != nil {
		return err
	}
	if !r.protection.IsProtectedRoot(record.RootID) {
		return fmt.Errorf(
			"%w: MCP server is not in a protected Root",
			model.ErrProtected,
		)
	}
	if record.Revision != expectedArtifactRevision {
		return model.ErrConflict
	}

	return r.protectedOverlay.Delete(
		ctx,
		ref,
		InstallationNamespace,
		expectedArtifactRevision,
		expectedOverlayRevision,
	)
}

func (r *ArtifactOverlayRepository) PurgeServerLocalState(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if r == nil || r.localState == nil {
		return model.ErrClosed
	}
	return r.localState.PurgeArtifactLocalState(ctx, ref)
}

func (r *ArtifactOverlayRepository) artifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if r == nil ||
		r.artifacts == nil ||
		r.protection == nil ||
		r.protectedOverlay == nil ||
		r.localState == nil {
		return artifact.Artifact{}, model.ErrClosed
	}
	return r.artifacts.Get(ctx, ref)
}

type serverOverlayPayload struct {
	ServerData mcpDomainServer.ServerData `json:"serverData"`
}

func encodeProtectedServerOverlay(
	value ServerOverlay,
) (json.RawMessage, error) {
	return jsonutil.MarshalCanonicalObject(
		serverOverlayPayload{
			ServerData: value.ServerData.Clone(),
		},
		model.MaxLocalDataBytes,
	)
}

func decodeProtectedServerOverlay(
	record artifactOverlay.Record,
) (ServerOverlay, error) {
	var payload serverOverlayPayload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&payload,
		model.MaxLocalDataBytes,
	); err != nil {
		return ServerOverlay{}, fmt.Errorf(
			"%w: decode protected MCP installation overlay: %w",
			model.ErrInvalid,
			err,
		)
	}

	value := ServerOverlay{
		SchemaVersion: record.SchemaVersion,
		Revision:      record.Revision,
		ServerData:    payload.ServerData.Clone(),
	}
	if err := value.Validate(); err != nil {
		return ServerOverlay{}, err
	}
	return value, nil
}

func validateOverlayTransition(
	expected uint64,
	next uint64,
) error {
	if expected == ^uint64(0) ||
		next == 0 ||
		next != expected+1 {
		return fmt.Errorf(
			"%w: invalid MCP installation overlay revision transition",
			model.ErrInvalid,
		)
	}
	return nil
}

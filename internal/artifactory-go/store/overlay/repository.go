package overlay

import (
	"context"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
)

// ProtectedRepository owns protected Artifact overlay persistence. DeleteOverlay
// atomically detaches namespace bindings and queues their physical secret refs.
type ProtectedRepository interface {
	GetOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		namespace overlayModel.Namespace,
	) (overlayModel.Record, bool, error)
	PutOverlay(ctx context.Context, request overlayModel.PutRequest, now time.Time) (overlayModel.Record, error)
	DeleteOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		namespace overlayModel.Namespace,
		expectedArtifactRevision, expectedOverlayRevision uint64,
		now time.Time,
	) error
}

// StoreRepository owns store-scoped non-secret overlay persistence.
type StoreRepository interface {
	GetStoreOverlay(ctx context.Context, namespace overlayModel.Namespace) (overlayModel.StoreRecord, bool, error)
	PutStoreOverlay(
		ctx context.Context,
		request overlayModel.StorePutRequest,
		now time.Time,
	) (overlayModel.StoreRecord, error)
	DeleteStoreOverlay(ctx context.Context, namespace overlayModel.Namespace, expectedRevision uint64) error
}

// Repository groups only Overlay-owned persistence responsibilities.
type Repository interface {
	ProtectedRepository
	StoreRepository
}

package internal

import (
	"context"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
)

// ArtifactReader is intentionally narrow. Local state is attached to exact
// Artifact refs but does not own source discovery, definitions, packages, or
// source lifecycle.
type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)
}

// AttachBindingRequest describes the SQLite transaction that publishes a
// staged physical secret value as the active ref for one binding.
type AttachBindingRequest struct {
	Key secretModel.BindingKey

	ExpectedArtifactRevision uint64
	ExpectedBindingRevision  uint64

	Record secretModel.Record
}

// Repository is the metadata persistence port for protected overlays and
// Artifact-local secret bindings.
//
// Every operation that detaches a ref must atomically enqueue physical cleanup
// before deleting the metadata reference.
type Repository interface {
	GetOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		namespace overlayModel.Namespace,
	) (overlayModel.Record, bool, error)

	GetStoreOverlay(
		ctx context.Context,
		namespace overlayModel.Namespace,
	) (overlayModel.StoreRecord, bool, error)

	PutStoreOverlay(
		ctx context.Context,
		request overlayModel.StorePutRequest,
		now time.Time,
	) (overlayModel.StoreRecord, error)

	DeleteStoreOverlay(
		ctx context.Context,
		namespace overlayModel.Namespace,
		expectedRevision uint64,
	) error

	PutOverlay(
		ctx context.Context,
		request overlayModel.PutRequest,
		now time.Time,
	) (overlayModel.Record, error)

	DeleteOverlay(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		namespace overlayModel.Namespace,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
		now time.Time,
	) error

	GetBinding(
		ctx context.Context,
		key secretModel.BindingKey,
	) (secretModel.Binding, bool, error)

	CreatePendingSecret(
		ctx context.Context,
		record secretModel.Record,
	) error

	AttachSecretBinding(
		ctx context.Context,
		request AttachBindingRequest,
		now time.Time,
	) (secretModel.Binding, error)

	ClearSecretBinding(
		ctx context.Context,
		request secretModel.ClearBindingRequest,
		now time.Time,
	) error

	QueueSecretForCleanup(
		ctx context.Context,
		ref secretModel.Ref,
		now time.Time,
	) error

	PurgeArtifactLocalState(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		now time.Time,
	) error

	RecoverPendingSecrets(
		ctx context.Context,
		now time.Time,
	) error

	ListSecretCleanup(
		ctx context.Context,
		maximum int,
	) ([]secretModel.Cleanup, error)

	CompleteSecretCleanup(
		ctx context.Context,
		ref secretModel.Ref,
	) error

	RecordSecretCleanupFailure(
		ctx context.Context,
		ref secretModel.Ref,
		reason string,
		now time.Time,
	) error
}

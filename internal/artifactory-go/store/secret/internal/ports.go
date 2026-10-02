package internal

import (
	"context"
	"time"

	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlay "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	secret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
)

// ArtifactReader is intentionally narrow. Local state is attached to exact
// Artifact refs but does not own source discovery, definitions, packages, or
// source lifecycle.
type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)
}

// AttachBindingRequest describes the SQLite transaction that publishes a
// staged physical secret value as the active ref for one binding.
type AttachBindingRequest struct {
	Key secret.BindingKey

	ExpectedArtifactRevision uint64
	ExpectedBindingRevision  uint64

	Record secret.Record
}

// Repository is the metadata persistence port for protected overlays and
// Artifact-local secret bindings.
//
// Every operation that detaches a ref must atomically enqueue physical cleanup
// before deleting the metadata reference.
type Repository interface {
	GetOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		namespace overlay.Namespace,
	) (overlay.Record, bool, error)

	GetStoreOverlay(
		ctx context.Context,
		namespace overlay.Namespace,
	) (overlay.StoreRecord, bool, error)

	PutStoreOverlay(
		ctx context.Context,
		request overlay.StorePutRequest,
		now time.Time,
	) (overlay.StoreRecord, error)

	DeleteStoreOverlay(
		ctx context.Context,
		namespace overlay.Namespace,
		expectedRevision uint64,
	) error

	PutOverlay(
		ctx context.Context,
		request overlay.PutRequest,
		now time.Time,
	) (overlay.Record, error)

	DeleteOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		namespace overlay.Namespace,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
		now time.Time,
	) error

	GetBinding(
		ctx context.Context,
		key secret.BindingKey,
	) (secret.Binding, bool, error)

	CreatePendingSecret(
		ctx context.Context,
		record secret.Record,
	) error

	AttachSecretBinding(
		ctx context.Context,
		request AttachBindingRequest,
		now time.Time,
	) (secret.Binding, error)

	ClearSecretBinding(
		ctx context.Context,
		request secret.ClearBindingRequest,
		now time.Time,
	) error

	QueueSecretForCleanup(
		ctx context.Context,
		ref secret.Ref,
		now time.Time,
	) error

	PurgeArtifactLocalState(
		ctx context.Context,
		ref artifact.ArtifactRef,
		now time.Time,
	) error

	RecoverPendingSecrets(
		ctx context.Context,
		now time.Time,
	) error

	ListSecretCleanup(
		ctx context.Context,
		maximum int,
	) ([]secret.Cleanup, error)

	CompleteSecretCleanup(
		ctx context.Context,
		ref secret.Ref,
	) error

	RecordSecretCleanupFailure(
		ctx context.Context,
		ref secret.Ref,
		reason string,
		now time.Time,
	) error
}

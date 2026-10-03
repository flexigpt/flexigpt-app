package secret

import (
	"context"
	"time"

	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
)

type AttachBindingRequest struct {
	Key secretModel.BindingKey

	ExpectedArtifactRevision uint64
	ExpectedBindingRevision  uint64

	Record secretModel.Record
}

type Repository interface {
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

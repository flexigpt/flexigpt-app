package overlay

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
)

type API interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		namespace overlayModel.Namespace,
	) (overlayModel.Record, bool, error)

	Put(
		ctx context.Context,
		request overlayModel.PutRequest,
	) (overlayModel.Record, error)

	Delete(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		namespace overlayModel.Namespace,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
	) error
}

type StoreAPI interface {
	GetStoreOverlay(
		ctx context.Context,
		namespace overlayModel.Namespace,
	) (overlayModel.StoreRecord, bool, error)

	PutStoreOverlay(
		ctx context.Context,
		request overlayModel.StorePutRequest,
	) (overlayModel.StoreRecord, error)

	DeleteStoreOverlay(
		ctx context.Context,
		namespace overlayModel.Namespace,
		expectedRevision uint64,
	) error
}

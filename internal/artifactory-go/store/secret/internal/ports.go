package internal

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
)

type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)
}

type AttachBindingRequest = secret.AttachBindingRequest

type Repository interface {
	overlay.Repository
	secret.Repository
	artifactcleanupFlow.Repository
}

package artifact

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type ArtifactIDProvider interface {
	NewArtifactID(ctx context.Context) (basespec.ArtifactID, error)
}

type ArtifactIDProviderFunc func(
	context.Context,
) (basespec.ArtifactID, error)

func (f ArtifactIDProviderFunc) NewArtifactID(
	ctx context.Context,
) (basespec.ArtifactID, error) {
	return f(ctx)
}

func UUIDArtifactIDProvider() ArtifactIDProvider {
	generator := uuidutil.UUIDv7Generator{}
	return ArtifactIDProviderFunc(
		func(ctx context.Context) (basespec.ArtifactID, error) {
			id, err := generator.NewID(ctx)
			return basespec.ArtifactID(id), err
		},
	)
}

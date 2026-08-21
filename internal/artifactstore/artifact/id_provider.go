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
	return ArtifactIDProviderFunc(
		func(ctx context.Context) (basespec.ArtifactID, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			id := uuidutil.NewUUIDv7()
			return basespec.ArtifactID(id), nil
		},
	)
}

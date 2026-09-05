package artifactid

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

// Provider creates Store-owned Artifact IDs for automatic adoption.
//
// Providers never receive this capability. Explicit Artifact creation remains
// caller-ID-based, while reconciliation-created observed Artifacts receive IDs
// only from Artifact Store.
type Provider interface {
	NewArtifactID(ctx context.Context) (basespec.ArtifactID, error)
}

type ProviderFunc func(
	context.Context,
) (basespec.ArtifactID, error)

func (f ProviderFunc) NewArtifactID(
	ctx context.Context,
) (basespec.ArtifactID, error) {
	return f(ctx)
}

func NewUUIDProvider() Provider {
	return ProviderFunc(
		func(ctx context.Context) (basespec.ArtifactID, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return basespec.ArtifactID(uuidutil.NewUUIDv7()), nil
		},
	)
}

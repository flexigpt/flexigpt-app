package artifactstore

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/resource"
)

type (
	ResolveOptions   = resource.ResolveOptions
	ResolvedArtifact = resource.ResolvedArtifact
	VerifiedEntry    = resource.VerifiedEntry
)

// ResourceResolver is the consumer-facing Artifact Store capability for
// current resource resolution and controlled Source access.
type ResourceResolver interface {
	ResolveArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		options ResolveOptions,
	) (ResolvedArtifact, error)

	ResolveVerifiedLocalPath(
		ctx context.Context,
		resolved ResolvedArtifact,
		localLocator basespec.Locator,
	) (string, error)

	ReadCollectionEntry(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		locator basespec.Locator,
		maximumBytes int64,
	) (VerifiedEntry, error)

	ResolveSourceLocalPath(
		ctx context.Context,
		rootID basespec.RootID,
		sourceID basespec.SourceID,
		locator basespec.Locator,
	) (string, error)

	SupportsLocalPath(kind basespec.SourceKind) bool
}

var _ ResourceResolver = (*API)(nil)

func (a *API) ResolveArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	options ResolveOptions,
) (ResolvedArtifact, error) {
	if err := a.check(ctx); err != nil {
		return ResolvedArtifact{}, err
	}
	if a.resources == nil {
		return ResolvedArtifact{}, basespec.ErrClosed
	}
	return a.resources.ResolveArtifact(ctx, ref, options)
}

func (a *API) ResolveVerifiedLocalPath(
	ctx context.Context,
	resolved ResolvedArtifact,
	localLocator basespec.Locator,
) (string, error) {
	if err := a.check(ctx); err != nil {
		return "", err
	}
	if a.resources == nil {
		return "", basespec.ErrClosed
	}
	return a.resources.ResolveVerifiedLocalPath(
		ctx,
		resolved,
		localLocator,
	)
}

func (a *API) ReadCollectionEntry(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	locator basespec.Locator,
	maximumBytes int64,
) (VerifiedEntry, error) {
	if err := a.check(ctx); err != nil {
		return VerifiedEntry{}, err
	}
	if a.resources == nil {
		return VerifiedEntry{}, basespec.ErrClosed
	}
	return a.resources.ReadCollectionEntry(
		ctx,
		ref,
		sourceID,
		locator,
		maximumBytes,
	)
}

func (a *API) ResolveSourceLocalPath(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	locator basespec.Locator,
) (string, error) {
	if err := a.check(ctx); err != nil {
		return "", err
	}
	if a.resources == nil {
		return "", basespec.ErrClosed
	}
	return a.resources.ResolveSourceLocalPath(
		ctx,
		rootID,
		sourceID,
		locator,
	)
}

func (a *API) SupportsLocalPath(kind basespec.SourceKind) bool {
	return a != nil &&
		a.resources != nil &&
		a.resources.SupportsLocalPath(kind)
}

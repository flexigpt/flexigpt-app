package artifactstore

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/resource"

	artifactAPI "github.com/flexigpt/flexigpt-app/internal/artifactstore/api"
)

func (a *API) ResolveArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
	options artifactAPI.ResolveOptions,
) (artifactAPI.ResolvedArtifact, error) {
	if err := a.check(ctx); err != nil {
		return artifactAPI.ResolvedArtifact{}, err
	}
	if a.resources == nil {
		return artifactAPI.ResolvedArtifact{}, basespec.ErrClosed
	}
	value, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{
			VerifySourceContent: options.VerifySourceContent,
		},
	)
	if err != nil {
		return artifactAPI.ResolvedArtifact{}, err
	}
	return resolvedArtifactForAPI(value), nil
}

func (a *API) ResolveVerifiedLocalPath(
	ctx context.Context,
	resolved artifactAPI.ResolvedArtifact,
	localLocator basespec.Locator,
) (string, error) {
	if err := a.check(ctx); err != nil {
		return "", err
	}
	if a.resources == nil {
		return "", basespec.ErrClosed
	}
	if err := resolved.Validate(); err != nil {
		return "", err
	}
	return a.resources.ResolveVerifiedLocalPath(
		ctx,
		resolvedArtifactForStore(resolved),
		localLocator,
	)
}

func (a *API) ReadCollectionEntry(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	locator basespec.Locator,
	maximumBytes int64,
) (artifactAPI.VerifiedEntry, error) {
	if err := a.check(ctx); err != nil {
		return artifactAPI.VerifiedEntry{}, err
	}
	if a.resources == nil {
		return artifactAPI.VerifiedEntry{}, basespec.ErrClosed
	}
	value, err := a.resources.ReadCollectionEntry(
		ctx,
		ref,
		sourceID,
		locator,
		maximumBytes,
	)
	if err != nil {
		return artifactAPI.VerifiedEntry{}, err
	}
	return verifiedEntryForAPI(value), nil
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

func resolvedArtifactForAPI(
	value resource.ResolvedArtifact,
) artifactAPI.ResolvedArtifact {
	return artifactAPI.ResolvedArtifact{
		Artifact:         value.Artifact.Clone(),
		Collection:       value.Collection.Clone(),
		Definition:       value.Definition.Clone(),
		Occurrence:       catalog.CloneOccurrence(value.Occurrence),
		Source:           artifactAPI.CloneSourceSummary(value.Source),
		CatalogRevision:  value.CatalogRevision,
		SourceGeneration: value.SourceGeneration,
	}.Clone()
}

func resolvedArtifactForStore(
	value artifactAPI.ResolvedArtifact,
) resource.ResolvedArtifact {
	value = value.Clone()
	return resource.ResolvedArtifact{
		Artifact:         value.Artifact.Clone(),
		Collection:       value.Collection.Clone(),
		Definition:       value.Definition.Clone(),
		Occurrence:       catalog.CloneOccurrence(value.Occurrence),
		Source:           artifactAPI.CloneSourceSummary(value.Source),
		CatalogRevision:  value.CatalogRevision,
		SourceGeneration: value.SourceGeneration,
	}
}

func verifiedEntryForAPI(
	value resource.VerifiedEntry,
) artifactAPI.VerifiedEntry {
	return artifactAPI.VerifiedEntry{
		Collection:       value.Collection,
		SourceID:         value.SourceID,
		CatalogRevision:  value.CatalogRevision,
		SourceRevision:   value.SourceRevision,
		SourceGeneration: value.SourceGeneration,
		Content:          append([]byte(nil), value.Content...),
		Digest:           value.Digest,
	}.Clone()
}

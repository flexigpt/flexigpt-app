package collection

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

type CollectionCapabilityOccurrence = resolve.CapabilityOccurrence

type CollectionCapabilityPlan struct {
	Collection  CollectionView                   `json:"collection"`
	Occurrences []CollectionCapabilityOccurrence `json:"occurrences"`
	Complete    bool                             `json:"complete"`
}

func (a *API) ResolveCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CollectionCapabilityPlan, error) {
	if a == nil || a.resolver == nil {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Collection resolver is unavailable",
			basespec.ErrUnsupported,
		)
	}

	generic, err := a.resolver.ResolvePluginCapabilities(ctx, ref)
	if err != nil {
		return CollectionCapabilityPlan{}, err
	}
	if generic.RootType != declaration.TypePlugin {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Artifact %q is not a Plugin",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	if generic.RootArtifact == nil {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Collection has no source-backed Artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	view, err := a.Read(ctx, *generic.RootArtifact)
	if err != nil {
		return CollectionCapabilityPlan{}, err
	}

	return CollectionCapabilityPlan{
		Collection: view,
		Occurrences: append(
			[]CollectionCapabilityOccurrence(nil),
			generic.Occurrences...,
		),
		Complete: generic.Complete,
	}, nil
}

package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
)

func (a *API) ResolveEnabledTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedToolView, error) {
	if err := a.ready(ctx); err != nil {
		return ResolvedToolView{}, err
	}

	return consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (ResolvedToolView, error) {
			value, err := a.getTool(sessionCtx, ref)
			if err != nil {
				return ResolvedToolView{}, err
			}
			collectionView, err := a.collectionForTool(
				sessionCtx,
				value.Artifact.LogicalName,
			)
			if err != nil {
				return ResolvedToolView{}, err
			}

			output := ResolvedToolView{
				Tool:       toolView(value),
				Collection: collectionView,
			}
			if !output.Enabled() {
				return ResolvedToolView{}, fmt.Errorf(
					"%w: Tool %q or its Collection is disabled",
					basespec.ErrReferenceUnresolved,
					value.Artifact.LogicalName,
				)
			}

			if err := a.verifyCurrentArtifact(
				sessionCtx,
				value.Artifact,
			); err != nil {
				return ResolvedToolView{}, err
			}
			if err := a.verifyCurrentArtifact(
				sessionCtx,
				collectionView.Artifact,
			); err != nil {
				return ResolvedToolView{}, err
			}
			return output, nil
		},
	)
}

func (a *API) verifyCurrentArtifact(
	ctx context.Context,
	record artifact.Artifact,
) error {
	if record.ResolvedDefinition == nil ||
		record.SourceContentDigest == nil {
		return fmt.Errorf(
			"%w: Tool catalog Artifact lacks resolved source identity",
			basespec.ErrReferenceUnresolved,
		)
	}

	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return err
	}
	if !sourceValue.Enabled {
		return fmt.Errorf(
			"%w: built-in Tool Source is disabled",
			basespec.ErrReferenceUnresolved,
		)
	}

	resolved, err := a.resources.ResolveArtifact(
		ctx,
		record.Ref(),
		resource.ResolveOptions{},
	)
	if err != nil {
		return err
	}
	if err := resolved.Validate(); err != nil {
		return err
	}
	if resolved.Artifact.Ref() != record.Ref() ||
		resolved.Artifact.Revision != record.Revision ||
		resolved.Artifact.Binding != record.Binding ||
		resolved.Definition.Digest != *record.ResolvedDefinition ||
		resolved.Artifact.SourceContentDigest == nil ||
		*resolved.Artifact.SourceContentDigest != *record.SourceContentDigest {
		return fmt.Errorf(
			"%w: Tool catalog Artifact changed during verification",
			basespec.ErrRefreshRequired,
		)
	}
	return nil
}

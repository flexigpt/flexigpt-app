package refresh

import (
	"context"
	"errors"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func EnsureSourceCurrent(
	ctx context.Context,
	refreshes API,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) error {
	inspection, err := refreshes.InspectSource(ctx, rootID, sourceID)
	if err == nil && inspection.IsCurrent() {
		return nil
	}
	if err != nil &&
		!errors.Is(err, spec.ErrRefreshStateNotFound) {
		return err
	}

	_, err = refreshes.RefreshSource(ctx, rootID, sourceID)
	return err
}

type DiscoveryReconciler func(
	current sourceModel.DiscoverySpec,
	desired sourceModel.DiscoverySpec,
) sourceModel.DiscoverySpec

type EnsureAndRefreshSourceRequest struct {
	RootID rootModel.RootID
	Draft  sourceModel.Draft

	ReconcileDiscovery DiscoveryReconciler
}

func EnsureAndRefreshSource(
	ctx context.Context,
	sources source.API,
	refreshes API,
	request EnsureAndRefreshSourceRequest,
) (sourceModel.Summary, error) {
	if sources == nil || refreshes == nil {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: Source lifecycle dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	if err := request.RootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}

	draft := request.Draft
	if !draft.Enabled || draft.Discovery.Empty() {
		return sourceModel.Summary{}, fmt.Errorf(
			"%w: Source lifecycle requires an enabled Source with discovery",
			spec.ErrInvalid,
		)
	}
	draft.Discovery = draft.Discovery.Normalized()
	if err := draft.Discovery.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}

	summary, _, err := sources.Ensure(ctx, request.RootID, draft)
	if err != nil {
		return sourceModel.Summary{}, err
	}

	desired := draft.Discovery
	if request.ReconcileDiscovery != nil {
		desired = request.ReconcileDiscovery(
			summary.Discovery.Clone(),
			desired,
		)
	}
	desired = desired.Normalized()
	if err := desired.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}

	if !summary.Enabled ||
		summary.DisplayName != draft.DisplayName ||
		!summary.Discovery.Equal(desired) {
		summary, err = sources.Update(
			ctx,
			request.RootID,
			summary.ID,
			sourceModel.Update{
				ExpectedRevision: summary.Revision,
				DisplayName:      draft.DisplayName,
				Enabled:          true,
				Discovery:        &desired,
			},
		)
		if err != nil {
			return sourceModel.Summary{}, err
		}
	}

	if _, err := refreshes.RefreshSource(
		ctx,
		request.RootID,
		summary.ID,
	); err != nil {
		return sourceModel.Summary{}, err
	}
	return summary, nil
}

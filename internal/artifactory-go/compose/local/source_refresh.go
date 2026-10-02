package local

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

// EnsureSourceCurrent refreshes one Source only when no refresh state exists
// or its current state differs from the Source and decoder configuration.
func EnsureSourceCurrent(
	ctx context.Context,
	discovery DiscoveryAPI,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if discovery == nil {
		return fmt.Errorf(
			"%w: Source discovery API is nil",
			model.ErrInvalid,
		)
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Source refresh context is nil",
			model.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if err := sourceID.Validate(); err != nil {
		return err
	}

	inspection, err := discovery.InspectSource(ctx, rootID, sourceID)
	if err == nil && inspection.IsCurrent() {
		return nil
	}
	if err != nil &&
		!errors.Is(err, model.ErrRefreshStateNotFound) {
		return err
	}

	_, err = discovery.RefreshSource(ctx, rootID, sourceID)
	return err
}

package collection

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// EnsureManagedDeclarationDiscovery prepares the caller-owned managed
// declaration universe. Package publication and refresh remain separate,
// explicit operations owned by ManagePackage.
func (a *API) EnsureManagedDeclarationDiscovery(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator spec.Locator,
	requiredDecoder spec.DecoderID,
) (sourceModel.Summary, error) {
	if err := a.requireDeclarationAuthoring(); err != nil {
		return sourceModel.Summary{}, err
	}
	if ctx == nil {
		return sourceModel.Summary{}, fmt.Errorf("%w: declaration discovery context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := rootID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}
	// Validate the complete external request before managedSource can repair
	// domain Source enablement or metadata.
	if err := locator.Validate(false); err != nil {
		return sourceModel.Summary{}, err
	}
	if err := requiredDecoder.Validate(); err != nil {
		return sourceModel.Summary{}, err
	}

	current, err := a.managedSource(ctx, rootID, sourceID)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	next, err := source.AddDeclarationDiscovery(
		current.Discovery,
		locator,
		requiredDecoder,
		true,
	)
	if err != nil {
		return sourceModel.Summary{}, err
	}
	if current.Discovery.Equal(next) {
		return current, nil
	}
	updated, err := a.sources.Update(ctx, rootID, sourceID, sourceModel.Update{
		ExpectedRevision: current.Revision,
		DisplayName:      current.DisplayName,
		Enabled:          current.Enabled,
		Discovery:        &next,
	})
	if err != nil {
		return sourceModel.Summary{}, fmt.Errorf("update managed declaration discovery: %w", err)
	}
	return updated, nil
}

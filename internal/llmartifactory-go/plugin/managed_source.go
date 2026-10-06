package plugin

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
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
	updated, err := a.sources.PrepareDiscovery(
		ctx,
		rootID,
		sourceID,
		sourceModel.DiscoveryPreparation{
			ExpectedRevision: current.Revision,
			Intent:           sourceModel.DiscoveryPreparationAdditive,
			Requirement: sourceModel.DiscoveryRequirement{
				ExplicitLocators: []spec.Locator{locator},
				DecoderHints: []sourceModel.DecoderHint{{
					Locator:    locator,
					DecoderIDs: []spec.DecoderID{requiredDecoder},
				}},
				RequireAuthoritative: true,
			},
		},
	)
	if err != nil {
		return sourceModel.Summary{}, fmt.Errorf(
			"prepare managed declaration discovery: %w",
			err,
		)
	}
	return updated, nil
}

package internal

import (
	"context"
	"fmt"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// EnsureProtectedTopology creates or verifies a declared protected Root and
// its generic Sources. Feature installers remain responsible for declaration
// contracts, package validation, package publication, and Source refresh.
func (c *Service) EnsureProtectedTopology(
	ctx context.Context,
	declaration installModel.Declaration,
) (installModel.Installed, error) {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return installModel.Installed{}, err
	}
	if err := declaration.Validate(); err != nil {
		return installModel.Installed{}, err
	}
	if c.rootMutationPolicy == nil ||
		!c.rootMutationPolicy.IsProtectedRoot(declaration.Root.ID) {
		return installModel.Installed{}, fmt.Errorf(
			"%w: declared Root %q is not protected by application policy",
			spec.ErrProtected,
			declaration.Root.ID,
		)
	}

	rootValue, err := c.rootSystem.EnsureSystem(ctx, declaration.Root)
	if err != nil {
		return installModel.Installed{}, err
	}

	output := installModel.Installed{
		Root:    rootValue,
		Sources: make([]sourceModel.Summary, 0, len(declaration.Sources)),
	}
	for _, draft := range declaration.Sources {
		value, _, err := c.Sources.Ensure(ctx, rootValue.ID, draft)
		if err != nil {
			return installModel.Installed{}, err
		}

		// The protected topology declaration owns the required Source
		// baseline. Package installers may have additively prepared further
		// declaration discovery, so topology reconciliation must not erase
		// those valid package-owned scopes.
		desiredDiscovery := source.MergeDiscoveryScopes(
			value.Discovery,
			draft.Discovery,
		)
		if value.DisplayName != draft.DisplayName ||
			value.Enabled != draft.Enabled ||
			!value.Discovery.Equal(desiredDiscovery) {
			value, err = c.Sources.Update(
				ctx,
				rootValue.ID,
				value.ID,
				sourceModel.Update{
					ExpectedRevision: value.Revision,
					DisplayName:      draft.DisplayName,
					Enabled:          draft.Enabled,
					Discovery:        &desiredDiscovery,
				},
			)
			if err != nil {
				return installModel.Installed{}, err
			}
		}

		output.Sources = append(output.Sources, value)
	}
	return output, nil
}

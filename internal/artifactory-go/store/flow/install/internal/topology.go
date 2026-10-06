package internal

import (
	"context"
	"fmt"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
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
		value, err := c.Sources.Create(ctx, rootValue.ID, draft)
		if err != nil {
			return installModel.Installed{}, err
		}
		output.Sources = append(output.Sources, value)
	}
	return output, nil
}

package system

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
)

// EnsureProtectedTopology creates or verifies a declared protected Root and
// its generic Sources. Feature installers remain responsible for declaration
// contracts, package validation, package publication, and Source refresh.
func (c *Components) EnsureProtectedTopology(
	ctx context.Context,
	declaration topology.Declaration,
) (topology.Installed, error) {
	if c == nil || c.Roots == nil || c.Sources == nil {
		return topology.Installed{}, basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return topology.Installed{}, err
	}
	if err := declaration.Validate(); err != nil {
		return topology.Installed{}, err
	}
	if c.rootMutationPolicy == nil ||
		!c.rootMutationPolicy.IsProtectedRoot(declaration.Root.ID) {
		return topology.Installed{}, fmt.Errorf(
			"%w: declared Root %q is not protected by application policy",
			basespec.ErrProtected,
			declaration.Root.ID,
		)
	}

	rootValue, err := c.Roots.EnsureSystem(ctx, declaration.Root)
	if err != nil {
		return topology.Installed{}, err
	}

	output := topology.Installed{
		Root:    rootValue,
		Sources: make([]source.Summary, 0, len(declaration.Sources)),
	}
	for _, draft := range declaration.Sources {
		value, err := c.Sources.Create(ctx, rootValue.ID, draft)
		if err != nil {
			return topology.Installed{}, err
		}
		if !protectedSourceIntentMatches(value, rootValue.ID, draft) {
			return topology.Installed{}, fmt.Errorf(
				"%w: protected Source %q declaration differs from stored topology",
				basespec.ErrConflict,
				draft.ID,
			)
		}
		output.Sources = append(output.Sources, value)
	}
	return output, nil
}

func protectedSourceIntentMatches(
	value source.Summary,
	rootID root.RootID,
	draft source.Draft,
) bool {
	return value.ID == draft.ID &&
		value.RootID == rootID &&
		value.StorageKey == draft.StorageKey &&
		value.Kind == draft.Kind &&
		value.DisplayName == draft.DisplayName &&
		value.Enabled == draft.Enabled &&
		value.Discovery.Equal(draft.Discovery)
}

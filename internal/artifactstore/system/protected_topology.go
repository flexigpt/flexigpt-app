package system

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
)

// These aliases preserve the existing system package API for callers that
// have not yet moved to artifactstore/topology.
type (
	ProtectedTopologyDeclaration = topology.Declaration
	ProtectedTopology            = topology.Installed
)

// EnsureProtectedTopology creates or verifies a declared protected Root and
// its generic Sources. Feature installers remain responsible for feature
// Collections, Artifacts, package validation, and package publication.
func (c *Components) EnsureProtectedTopology(
	ctx context.Context,
	declaration topology.Declaration,
) (topology.Installed, error) {
	if c == nil || c.Roots == nil || c.Sources == nil {
		return topology.Installed{}, basespec.ErrClosed
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
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
	rootID basespec.RootID,
	draft source.Draft,
) bool {
	return value.ID == draft.ID &&
		value.RootID == rootID &&
		value.Kind == draft.Kind &&
		value.DisplayName == draft.DisplayName &&
		value.Enabled == draft.Enabled
}
